package svg

import (
	"strconv"
	"strings"
)

// pathCursor walks the token stream of an SVG path "d" attribute. It holds the
// state the smooth-curve commands need: the previous command, and the last
// control point of a cubic (S) or quadratic (T) curve.
type pathCursor struct {
	path          *svgPath
	tokens        []pathToken
	index         int
	command       byte
	previous      byte
	lastCubic     svgPoint
	lastQuadratic svgPoint
	relative      bool
}

func (path *svgPath) parse(data string) bool {
	cursor := &pathCursor{path: path, tokens: scanPathTokens(data)}
	if len(cursor.tokens) == 0 {
		return false
	}
	for cursor.index < len(cursor.tokens) {
		if !cursor.readCommand() {
			return false
		}
		if !cursor.apply() {
			return false
		}
		// The command may have been rewritten (M implies L for the pairs that
		// follow it), and the smooth-curve commands compare against that value.
		cursor.previous = cursor.command
	}
	return path.hasCurrent
}

// readCommand consumes a command token when the cursor is on one. A number token
// repeats the command in effect, which is only valid once one has been seen.
func (cursor *pathCursor) readCommand() bool {
	if cursor.tokens[cursor.index].command != 0 {
		cursor.command = cursor.tokens[cursor.index].command
		cursor.index++
	}
	if cursor.command == 0 {
		return false
	}
	cursor.relative = cursor.command >= 'a' && cursor.command <= 'z'
	return true
}

// apply dispatches the command in effect. Lower-case commands are relative and
// share the handler of their upper-case counterpart.
func (cursor *pathCursor) apply() bool {
	kind := cursor.command
	if kind >= 'a' && kind <= 'z' {
		kind -= 'a' - 'A'
	}
	switch kind {
	case 'M':
		return cursor.moveTo()
	case 'L':
		return cursor.lineTo()
	case 'H':
		return cursor.horizontalTo()
	case 'V':
		return cursor.verticalTo()
	case 'C':
		return cursor.cubeTo()
	case 'S':
		return cursor.smoothCubeTo()
	case 'Q':
		return cursor.quadTo()
	case 'T':
		return cursor.smoothQuadTo()
	case 'A':
		return cursor.arcTo()
	case 'Z':
		cursor.path.close()
		return true
	default:
		return false
	}
}

// read consumes count number tokens. It fails if a command token appears first,
// which means the path supplied too few coordinates for the command.
func (cursor *pathCursor) read(count int) ([]float64, bool) {
	if cursor.index+count > len(cursor.tokens) {
		return nil, false
	}
	values := make([]float64, count)
	for offset := range values {
		if cursor.tokens[cursor.index+offset].command != 0 {
			return nil, false
		}
		values[offset] = cursor.tokens[cursor.index+offset].number
	}
	cursor.index += count
	return values, true
}

// point resolves a coordinate pair, treating it as an offset from the current
// point for relative commands.
func (cursor *pathCursor) point(x, y float64) svgPoint {
	if cursor.relative && cursor.path.hasCurrent {
		return svgPoint{cursor.path.current.x + x, cursor.path.current.y + y}
	}
	return svgPoint{x, y}
}

func (cursor *pathCursor) moveTo() bool {
	values, ok := cursor.read(2)
	if !ok {
		return false
	}
	start := cursor.point(values[0], values[1])
	cursor.path.moveTo(start.x, start.y)
	// Further coordinate pairs after a moveto are implicit linetos.
	if cursor.relative {
		cursor.command = 'l'
	} else {
		cursor.command = 'L'
	}
	return true
}

func (cursor *pathCursor) lineTo() bool {
	values, ok := cursor.read(2)
	if !ok {
		return false
	}
	end := cursor.point(values[0], values[1])
	cursor.path.lineTo(end.x, end.y)
	return true
}

func (cursor *pathCursor) horizontalTo() bool {
	values, ok := cursor.read(1)
	if !ok || !cursor.path.hasCurrent {
		return false
	}
	x := values[0]
	if cursor.relative {
		x += cursor.path.current.x
	}
	cursor.path.lineTo(x, cursor.path.current.y)
	return true
}

func (cursor *pathCursor) verticalTo() bool {
	values, ok := cursor.read(1)
	if !ok || !cursor.path.hasCurrent {
		return false
	}
	y := values[0]
	if cursor.relative {
		y += cursor.path.current.y
	}
	cursor.path.lineTo(cursor.path.current.x, y)
	return true
}

func (cursor *pathCursor) cubeTo() bool {
	values, ok := cursor.read(6)
	if !ok {
		return false
	}
	control1 := cursor.point(values[0], values[1])
	control2 := cursor.point(values[2], values[3])
	end := cursor.point(values[4], values[5])
	cursor.path.cubeTo(control1, control2, end)
	cursor.lastCubic = control2
	return true
}

// smoothCubeTo handles S: the first control point is the reflection of the
// previous curve's second control point, but only when the previous command was
// itself a cubic.
func (cursor *pathCursor) smoothCubeTo() bool {
	values, ok := cursor.read(4)
	if !ok || !cursor.path.hasCurrent {
		return false
	}
	control1 := cursor.path.current
	if isCubicCommand(cursor.previous) {
		control1 = cursor.reflect(cursor.lastCubic)
	}
	control2 := cursor.point(values[0], values[1])
	end := cursor.point(values[2], values[3])
	cursor.path.cubeTo(control1, control2, end)
	cursor.lastCubic = control2
	return true
}

func (cursor *pathCursor) quadTo() bool {
	values, ok := cursor.read(4)
	if !ok {
		return false
	}
	control := cursor.point(values[0], values[1])
	end := cursor.point(values[2], values[3])
	cursor.path.quadTo(control, end)
	cursor.lastQuadratic = control
	return true
}

// smoothQuadTo handles T: the control point is the reflection of the previous
// curve's control point, but only when the previous command was a quadratic.
func (cursor *pathCursor) smoothQuadTo() bool {
	values, ok := cursor.read(2)
	if !ok || !cursor.path.hasCurrent {
		return false
	}
	control := cursor.path.current
	if isQuadraticCommand(cursor.previous) {
		control = cursor.reflect(cursor.lastQuadratic)
	}
	end := cursor.point(values[0], values[1])
	cursor.path.quadTo(control, end)
	cursor.lastQuadratic = control
	return true
}

func (cursor *pathCursor) arcTo() bool {
	values, ok := cursor.read(7)
	if !ok {
		return false
	}
	// Arc endpoint semantics are preserved; this compact renderer draws
	// the chord until elliptical-arc flattening is required by Paper.
	end := cursor.point(values[5], values[6])
	cursor.path.lineTo(end.x, end.y)
	return true
}

// reflect mirrors a control point through the current point.
func (cursor *pathCursor) reflect(control svgPoint) svgPoint {
	return svgPoint{
		2*cursor.path.current.x - control.x,
		2*cursor.path.current.y - control.y,
	}
}

func isCubicCommand(command byte) bool {
	return command == 'C' || command == 'c' || command == 'S' || command == 's'
}

func isQuadraticCommand(command byte) bool {
	return command == 'Q' || command == 'q' || command == 'T' || command == 't'
}

type pathToken struct {
	command byte
	number  float64
}

func isPathSeparator(c byte) bool {
	return strings.ContainsRune(" \t\r\n,", rune(c))
}

func isPathCommand(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func scanPathTokens(value string) []pathToken {
	var tokens []pathToken
	for index := 0; index < len(value); {
		switch {
		case isPathSeparator(value[index]):
			index++
		case isPathCommand(value[index]):
			tokens = append(tokens, pathToken{command: value[index]})
			index++
		default:
			end, hasDigits := scanPathNumber(value, index)
			if !hasDigits {
				// Not a number at all; step past it so scanning makes progress.
				index = end + 1
				continue
			}
			if number, err := strconv.ParseFloat(value[index:end], 64); err == nil {
				tokens = append(tokens, pathToken{number: number})
			}
			index = end
		}
	}
	return tokens
}

// scanDigits returns the offset just past the run of decimal digits at index.
func scanDigits(value string, index int) int {
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		index++
	}
	return index
}

// scanPathNumber returns the offset just past the number literal starting at
// index, and whether that literal contained any digits. SVG accepts compact
// forms such as "-.5" and "1e3", so the sign, fraction and exponent are each
// optional.
func scanPathNumber(value string, index int) (end int, hasDigits bool) {
	if index < len(value) && (value[index] == '+' || value[index] == '-') {
		index++
	}

	afterInteger := scanDigits(value, index)
	hasDigits = afterInteger > index
	index = afterInteger

	if index < len(value) && value[index] == '.' {
		afterFraction := scanDigits(value, index+1)
		hasDigits = hasDigits || afterFraction > index+1
		index = afterFraction
	}

	// An exponent only counts when digits follow it; otherwise the "e" belongs
	// to whatever comes next.
	if index < len(value) && (value[index] == 'e' || value[index] == 'E') {
		exponent := index + 1
		if exponent < len(value) && (value[exponent] == '+' || value[exponent] == '-') {
			exponent++
		}
		if afterExponent := scanDigits(value, exponent); afterExponent > exponent {
			index = afterExponent
		}
	}

	return index, hasDigits
}
