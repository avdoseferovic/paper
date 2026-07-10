package translate

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

const (
	transformFuncCalc       = "calc"
	transformFuncClamp      = "clamp"
	transformFuncMax        = "max"
	transformFuncMin        = "min"
	transformFuncRotate     = "rotate"
	transformFuncScale      = "scale"
	transformFuncScaleX     = "scalex"
	transformFuncScaleY     = "scaley"
	transformFuncTranslate  = "translate"
	transformFuncTranslateX = "translatex"
	transformFuncTranslateY = "translatey"
	transformFuncSkew       = "skew"
	transformFuncSkewX      = "skewx"
	transformFuncSkewY      = "skewy"

	transformOriginCenter  = "center"
	transformOriginDefault = transformOriginCenter + " " + transformOriginCenter
)

type transformRow struct {
	child     core.Row
	transform string
	origin    string
	fontSize  float64
}

func (r *transformRow) SetConfig(config *entity.Config) {
	if r.child != nil {
		r.child.SetConfig(config)
	}
}

func (r *transformRow) GetStructure() *node.Node[core.Structure] {
	ops := parseTransform(r.transform, r.fontSize, 0, 0)
	str := core.Structure{
		Type: "transform_row",
		Details: map[string]any{
			"transform":  r.transform,
			"origin":     transformOriginValue(r.origin),
			"operations": len(ops),
		},
	}
	n := node.New(str)
	if r.child != nil {
		n.AddNext(r.child.GetStructure())
	}
	return n
}

func (r *transformRow) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if r.child == nil {
		return 0
	}
	return r.child.GetHeight(provider, cell)
}

func (r *transformRow) Render(provider core.Provider, cell entity.Cell) {
	if r.child == nil {
		return
	}
	ops := parseTransform(r.transform, r.fontSize, cell.Width, cell.Height)
	transformProvider, ok := provider.(core.TransformProvider)
	if !ok || len(ops) == 0 {
		r.child.Render(provider, cell)
		return
	}
	box := cell.Copy()
	if box.Height <= 0 {
		box.Height = r.child.GetHeight(provider, &box)
	}
	originX, originY := parseTransformOrigin(r.origin, box.Width, box.Height, r.fontSize)
	transformProvider.WithTransform(&box, originX, originY, ops, func() {
		r.child.Render(provider, cell)
	})
}

func (r *transformRow) Add(cols ...core.Col) core.Row {
	if r.child != nil {
		r.child = r.child.Add(cols...)
	}
	return r
}

func (r *transformRow) WithStyle(style *props.Cell) core.Row {
	if r.child != nil {
		r.child = r.child.WithStyle(style)
	}
	return r
}

func (r *transformRow) GetColumns() []core.Col {
	if r.child == nil {
		return nil
	}
	return r.child.GetColumns()
}

func (r *transformRow) SplitAt(provider core.Provider, remainingHeight float64, width float64) (core.Row, core.Row, bool) {
	splittable, ok := r.child.(core.Splittable)
	if !ok {
		return nil, nil, false
	}
	first, rest, didSplit := splittable.SplitAt(provider, remainingHeight, width)
	if !didSplit {
		return nil, nil, false
	}
	return r.wrapSplit(first), r.wrapSplit(rest), true
}

func (r *transformRow) wrapSplit(row core.Row) core.Row {
	if row == nil {
		return nil
	}
	return &transformRow{
		child:     row,
		transform: r.transform,
		origin:    r.origin,
		fontSize:  r.fontSize,
	}
}

func applyTransform(style *css.ComputedStyle, rows []core.Row) []core.Row {
	if style == nil || len(rows) == 0 || len(parseTransform(style.Transform, style.FontSize, 0, 0)) == 0 {
		return rows
	}
	child := rows[0]
	if len(rows) > 1 {
		child = &combinedRow{rows: rows}
	}
	return []core.Row{&transformRow{
		child:     child,
		transform: strings.TrimSpace(style.Transform),
		origin:    style.TransformOrigin,
		fontSize:  style.FontSize,
	}}
}

func parseTransform(value string, fontSize, width, height float64) []props.TransformOp {
	remaining := strings.TrimSpace(strings.ToLower(value))
	if remaining == "" || remaining == cssValueNone {
		return nil
	}

	var ops []props.TransformOp
	for remaining != "" {
		open := strings.Index(remaining, "(")
		if open < 0 {
			break
		}
		name := strings.TrimSpace(remaining[:open])
		closeIdx := matchingCloseParen(remaining, open)
		if closeIdx < 0 {
			break
		}
		args := remaining[open+1 : closeIdx]
		remaining = strings.TrimSpace(remaining[closeIdx+1:])
		ops = appendTransformOp(ops, name, splitTransformArgs(args), fontSize, width, height)
	}
	return ops
}

func appendTransformOp(
	ops []props.TransformOp,
	name string,
	parts []string,
	fontSize, width, height float64,
) []props.TransformOp {
	switch name {
	case transformFuncRotate:
		return appendRotateTransform(ops, parts)
	case transformFuncScale, transformFuncScaleX, transformFuncScaleY:
		return appendScaleTransform(ops, name, parts)
	case transformFuncTranslate, transformFuncTranslateX, transformFuncTranslateY:
		return appendTranslateTransform(ops, name, parts, fontSize, width, height)
	case transformFuncSkew, transformFuncSkewX, transformFuncSkewY:
		return appendSkewTransform(ops, name, parts)
	default:
		return ops
	}
}

func appendRotateTransform(ops []props.TransformOp, parts []string) []props.TransformOp {
	if len(parts) == 0 {
		return ops
	}
	return append(ops, props.TransformOp{
		Type:   props.TransformRotate,
		Values: [2]float64{parseTransformAngle(parts[0]), 0},
	})
}

func appendScaleTransform(ops []props.TransformOp, name string, parts []string) []props.TransformOp {
	if len(parts) == 0 {
		return ops
	}
	switch name {
	case transformFuncScale:
		if len(parts) >= 2 {
			return append(ops, props.TransformOp{
				Type:   props.TransformScale,
				Values: [2]float64{parseTransformNumber(parts[0]), parseTransformNumber(parts[1])},
			})
		}
		scale := parseTransformNumber(parts[0])
		return append(ops, props.TransformOp{Type: props.TransformScale, Values: [2]float64{scale, scale}})
	case transformFuncScaleX:
		return append(ops, props.TransformOp{
			Type:   props.TransformScale,
			Values: [2]float64{parseTransformNumber(parts[0]), 1},
		})
	default:
		return append(ops, props.TransformOp{
			Type:   props.TransformScale,
			Values: [2]float64{1, parseTransformNumber(parts[0])},
		})
	}
}

func appendTranslateTransform(
	ops []props.TransformOp,
	name string,
	parts []string,
	fontSize, width, height float64,
) []props.TransformOp {
	if len(parts) == 0 {
		return ops
	}
	switch name {
	case transformFuncTranslate:
		x := parseTransformLength(parts[0], fontSize, width)
		y := 0.0
		if len(parts) >= 2 {
			y = parseTransformLength(parts[1], fontSize, height)
		}
		return append(ops, props.TransformOp{Type: props.TransformTranslate, Values: [2]float64{x, y}})
	case transformFuncTranslateX:
		return append(ops, props.TransformOp{
			Type:   props.TransformTranslate,
			Values: [2]float64{parseTransformLength(parts[0], fontSize, width), 0},
		})
	default:
		return append(ops, props.TransformOp{
			Type:   props.TransformTranslate,
			Values: [2]float64{0, parseTransformLength(parts[0], fontSize, height)},
		})
	}
}

func appendSkewTransform(ops []props.TransformOp, name string, parts []string) []props.TransformOp {
	if len(parts) == 0 {
		return ops
	}
	switch name {
	case transformFuncSkew:
		if len(parts) >= 2 {
			return append(ops, props.TransformOp{
				Type:   props.TransformSkew,
				Values: [2]float64{parseTransformAngle(parts[0]), parseTransformAngle(parts[1])},
			})
		}
		return append(ops, props.TransformOp{
			Type:   props.TransformSkewX,
			Values: [2]float64{parseTransformAngle(parts[0]), 0},
		})
	case transformFuncSkewX:
		return append(ops, props.TransformOp{
			Type:   props.TransformSkewX,
			Values: [2]float64{parseTransformAngle(parts[0]), 0},
		})
	default:
		return append(ops, props.TransformOp{
			Type:   props.TransformSkewY,
			Values: [2]float64{parseTransformAngle(parts[0]), 0},
		})
	}
}

func parseTransformOrigin(value string, width, height, fontSize float64) (float64, float64) {
	parts := splitTransformFields(strings.ToLower(strings.TrimSpace(value)))
	if len(parts) == 0 {
		return width / 2, height / 2
	}
	x := resolveTransformOriginComponent(parts[0], width, fontSize)
	if len(parts) == 1 {
		return x, height / 2
	}
	return x, resolveTransformOriginComponent(parts[1], height, fontSize)
}

func resolveTransformOriginComponent(value string, dimension, fontSize float64) float64 {
	switch strings.TrimSpace(value) {
	case "left", "top":
		return 0
	case transformOriginCenter:
		return dimension / 2
	case cssValueRight, "bottom":
		return dimension
	default:
		return parseTransformLength(value, fontSize, dimension)
	}
}

func transformOriginValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return transformOriginDefault
	}
	return strings.TrimSpace(value)
}

func parseTransformLength(value string, fontSize, context float64) float64 {
	value = strings.TrimSpace(strings.ToLower(value))
	name, args, ok := transformFunction(value)
	if !ok {
		return css.ParseLengthCtx(value, fontSize, context)
	}
	parts := splitTransformArgs(args)
	switch name {
	case transformFuncMin:
		if len(parts) == 0 {
			return 0
		}
		out := parseTransformLength(parts[0], fontSize, context)
		for _, part := range parts[1:] {
			out = math.Min(out, parseTransformLength(part, fontSize, context))
		}
		return out
	case transformFuncMax:
		if len(parts) == 0 {
			return 0
		}
		out := parseTransformLength(parts[0], fontSize, context)
		for _, part := range parts[1:] {
			out = math.Max(out, parseTransformLength(part, fontSize, context))
		}
		return out
	case transformFuncClamp:
		if len(parts) != 3 {
			return 0
		}
		lo := parseTransformLength(parts[0], fontSize, context)
		mid := parseTransformLength(parts[1], fontSize, context)
		hi := parseTransformLength(parts[2], fontSize, context)
		return math.Max(lo, math.Min(mid, hi))
	default:
		return css.ParseLengthCtx(value, fontSize, context)
	}
}

func parseTransformAngle(value string) float64 {
	value = strings.TrimSpace(strings.ToLower(value))
	name, args, ok := transformFunction(value)
	if ok {
		parts := splitTransformArgs(args)
		switch name {
		case transformFuncCalc:
			out, _ := evalTransformCalc(args, parseTransformAngle)
			return out
		case transformFuncMin:
			return minTransformValue(parts, parseTransformAngle)
		case transformFuncMax:
			return maxTransformValue(parts, parseTransformAngle)
		case transformFuncClamp:
			return clampTransformValue(parts, parseTransformAngle)
		}
	}
	return parseTransformAngleLeaf(value)
}

func parseTransformNumber(value string) float64 {
	value = strings.TrimSpace(strings.ToLower(value))
	name, args, ok := transformFunction(value)
	if ok {
		parts := splitTransformArgs(args)
		switch name {
		case transformFuncCalc:
			out, _ := evalTransformCalc(args, parseTransformNumber)
			return out
		case transformFuncMin:
			return minTransformValue(parts, parseTransformNumber)
		case transformFuncMax:
			return maxTransformValue(parts, parseTransformNumber)
		case transformFuncClamp:
			return clampTransformValue(parts, parseTransformNumber)
		}
	}
	out, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return out
}

func parseTransformAngleLeaf(value string) float64 {
	switch {
	case strings.HasSuffix(value, "turn"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(value, "turn"), 64)
		return v * 360
	case strings.HasSuffix(value, "grad"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(value, "grad"), 64)
		return v * 0.9
	case strings.HasSuffix(value, "rad"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(value, "rad"), 64)
		return v * 180 / math.Pi
	case strings.HasSuffix(value, "deg"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(value, "deg"), 64)
		return v
	default:
		v, _ := strconv.ParseFloat(value, 64)
		return v
	}
}

func minTransformValue(parts []string, parse func(string) float64) float64 {
	if len(parts) == 0 {
		return 0
	}
	out := parse(parts[0])
	for _, part := range parts[1:] {
		out = math.Min(out, parse(part))
	}
	return out
}

func maxTransformValue(parts []string, parse func(string) float64) float64 {
	if len(parts) == 0 {
		return 0
	}
	out := parse(parts[0])
	for _, part := range parts[1:] {
		out = math.Max(out, parse(part))
	}
	return out
}

func clampTransformValue(parts []string, parse func(string) float64) float64 {
	if len(parts) != 3 {
		return 0
	}
	lo := parse(parts[0])
	mid := parse(parts[1])
	hi := parse(parts[2])
	return math.Max(lo, math.Min(mid, hi))
}

func evalTransformCalc(value string, parseLeaf func(string) float64) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if idx, op := findTopLevelOperator(value, "+-"); idx >= 0 {
		left, ok := evalTransformCalc(value[:idx], parseLeaf)
		if !ok {
			return 0, false
		}
		right, ok := evalTransformCalc(value[idx+1:], parseLeaf)
		if !ok {
			return 0, false
		}
		if op == '+' {
			return left + right, true
		}
		return left - right, true
	}
	if idx, op := findTopLevelOperator(value, "*/"); idx >= 0 {
		left, ok := evalTransformCalc(value[:idx], parseLeaf)
		if !ok {
			return 0, false
		}
		right, ok := evalTransformCalc(value[idx+1:], parseLeaf)
		if !ok || op == '/' && right == 0 {
			return 0, false
		}
		if op == '*' {
			return left * right, true
		}
		return left / right, true
	}
	if hasOuterParens(value) {
		return evalTransformCalc(value[1:len(value)-1], parseLeaf)
	}
	return parseLeaf(value), true
}

func findTopLevelOperator(value, operators string) (int, byte) {
	depth := 0
	for i := len(value) - 1; i >= 0; i-- {
		switch value[i] {
		case ')':
			depth++
		case '(':
			depth--
		default:
			if depth == 0 && strings.ContainsRune(operators, rune(value[i])) && isBinaryTransformOperator(value, i) {
				return i, value[i]
			}
		}
	}
	return -1, 0
}

func isBinaryTransformOperator(value string, idx int) bool {
	if idx <= 0 {
		return false
	}
	if value[idx] != '+' && value[idx] != '-' {
		return true
	}
	for i := idx - 1; i >= 0; i-- {
		if unicode.IsSpace(rune(value[i])) {
			continue
		}
		return !strings.ContainsRune("+-*/(", rune(value[i]))
	}
	return false
}

func hasOuterParens(value string) bool {
	if !strings.HasPrefix(value, "(") || !strings.HasSuffix(value, ")") {
		return false
	}
	return matchingCloseParen(value, 0) == len(value)-1
}

func transformFunction(value string) (string, string, bool) {
	open := strings.Index(value, "(")
	if open <= 0 || !strings.HasSuffix(value, ")") {
		return "", "", false
	}
	closeIdx := matchingCloseParen(value, open)
	if closeIdx != len(value)-1 {
		return "", "", false
	}
	return strings.TrimSpace(value[:open]), value[open+1 : closeIdx], true
}

func matchingCloseParen(value string, open int) int {
	depth := 0
	for i := open; i < len(value); i++ {
		switch value[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitTransformArgs(value string) []string {
	parts := splitTransformCommas(value)
	if len(parts) <= 1 {
		parts = splitTransformFields(value)
	}
	out := parts[:0]
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func splitTransformCommas(value string) []string {
	var out []string
	depth := 0
	start := 0
	for i, r := range value {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(value[start:i]))
				start = i + len(string(r))
			}
		}
	}
	out = append(out, strings.TrimSpace(value[start:]))
	return out
}

func splitTransformFields(value string) []string {
	var out []string
	var b strings.Builder
	depth := 0
	flush := func() {
		token := strings.TrimSpace(b.String())
		if token != "" {
			out = append(out, token)
		}
		b.Reset()
	}
	for _, r := range value {
		switch {
		case r == '(':
			depth++
			b.WriteRune(r)
		case r == ')':
			if depth > 0 {
				depth--
			}
			b.WriteRune(r)
		case depth == 0 && unicode.IsSpace(r):
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return out
}
