package translate

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	goimage "image"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/avdoseferovic/paper/pkg/components/col"
	imagecomp "github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/net/html"
)

// dpiForRaster is the DPI used when rasterising SVG to PNG for PDF embedding.
const dpiForRaster = 150.0

// ImageResolver loads image bytes for a given <img src="…"> value. It returns
// the raw bytes and a hint extension ("png", "jpg", "svg", etc.). When err is
// non-nil the caller falls back to the <img>'s alt text.
type ImageResolver func(src string) (data []byte, ext string, err error)

// ErrImageResolverRefused is returned by the default resolver when asked to
// load a non-data: URI without an explicit html.WithImageBaseDir or the
// lower-level translate.WithImageResolver.
var ErrImageResolverRefused = errors.New(
	"html: default resolver refuses local file reads; configure html.WithImageBaseDir or translate.WithImageResolver",
)

// safeDefaultResolver only accepts data: URIs. It refuses any other src to
// prevent path-traversal attacks on user-controlled HTML.
func safeDefaultResolver(src string) ([]byte, string, error) {
	if strings.HasPrefix(src, "data:") {
		return decodeDataURI(src)
	}
	return nil, "", ErrImageResolverRefused
}

// decodeDataURI parses a data: URI of the form data:<mime>;base64,<payload>.
// Returns the decoded bytes and an extension hint inferred from the MIME type.
func decodeDataURI(uri string) ([]byte, string, error) {
	// data:<mediatype>;base64,<payload>
	prefix := strings.TrimPrefix(uri, "data:")
	commaIdx := strings.IndexByte(prefix, ',')
	if commaIdx < 0 {
		return nil, "", fmt.Errorf("html: invalid data URI")
	}
	header := prefix[:commaIdx]
	payload := prefix[commaIdx+1:]
	mediaType := header
	isB64 := false
	if semi := strings.IndexByte(header, ';'); semi >= 0 {
		mediaType = header[:semi]
		if strings.Contains(header[semi+1:], "base64") {
			isB64 = true
		}
	}
	var data []byte
	if isB64 {
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, "", fmt.Errorf("html: data URI base64 decode: %w", err)
		}
		data = decoded
	} else {
		data = []byte(payload)
	}
	ext := "png"
	switch mediaType {
	case "image/png":
		ext = "png"
	case "image/jpeg", "image/jpg":
		ext = "jpg"
	case "image/svg+xml":
		ext = "svg"
	}
	return data, ext, nil
}

// baseDirResolver returns a resolver that only loads files inside dir,
// rejecting any path that would escape via "../" or absolute prefix.
func baseDirResolver(dir string) ImageResolver {
	cleanBase, _ := filepath.Abs(filepath.Clean(dir))
	return func(src string) ([]byte, string, error) {
		if strings.HasPrefix(src, "data:") {
			return decodeDataURI(src)
		}
		// Reject absolute paths immediately.
		if filepath.IsAbs(src) {
			return nil, "", fmt.Errorf("html: absolute path %q refused outside base dir", src)
		}
		full, err := filepath.Abs(filepath.Join(cleanBase, src))
		if err != nil {
			return nil, "", err
		}
		if !strings.HasPrefix(full, cleanBase+string(filepath.Separator)) && full != cleanBase {
			return nil, "", fmt.Errorf("html: path %q escapes base dir", src)
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return nil, "", err
		}
		return data, extFromFilename(src), nil
	}
}

func extFromFilename(name string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	switch ext {
	case "jpeg":
		return "jpg"
	default:
		return ext
	}
}

// imageRow builds a block-level row for <img>. Returns the row and ok=true on
// success; ok=false signals the caller to fall back to alt text.
func (tr *translator) imageRow(n *dom.Node) (core.Row, bool) {
	src := tr.selectedImageSource(n)
	if src == "" {
		return nil, false
	}
	return tr.imageRowWithSource(n, src)
}

func (tr *translator) imageRowWithSource(n *dom.Node, src string) (core.Row, bool) {
	resolver := tr.imageResolver
	if resolver == nil {
		resolver = safeDefaultResolver
	}
	data, ext, err := resolver(src)
	if err != nil {
		tr.unsupported("img.src", err.Error())
		return nil, false
	}

	style := tr.imageStyle(n)
	dimensions := imageDimensions(n, style)
	intrinsicWidth, intrinsicHeight := 0.0, 0.0

	if ext == "svg" {
		pngBytes, w, h, rerr := rasteriseSVG(data, dimensions.width, dimensions.height)
		if rerr != nil {
			tr.unsupported("img.svg", rerr.Error())
			return nil, false
		}
		data = pngBytes
		ext = "png"
		intrinsicWidth = mmFromPx(w)
		intrinsicHeight = mmFromPx(h)
	} else {
		intrinsicWidth, intrinsicHeight = rasterImageSizeMM(data)
	}

	extType := extensionType(ext)
	if extType == "" {
		tr.unsupported("img.ext", "unsupported extension: "+ext)
		return nil, false
	}

	widthMM, heightMM := resolveImageDimensions(dimensions, intrinsicWidth, intrinsicHeight, 10)

	// Pick a small col that approximates the requested mm width. The image
	// fills the col (Percent=100, Center=true). Using a small col instead of
	// a full-width col + tiny Percent avoids the SVG getting visually squashed.
	cellWidth := tr.contentWidthMM
	if cellWidth <= 0 {
		cellWidth = 170.0
	}
	gridSize := tr.gridSize
	if gridSize <= 0 {
		gridSize = defaultGridSize
	}
	imgCols := imageCols(widthMM, cellWidth, gridSize)

	rect := props.Rect{Percent: 100, Center: true}
	if style != nil {
		rect.ObjectFit = style.ObjectFit
		rect.ObjectPosition = style.ObjectPosition
	}
	img := imagecomp.NewFromBytes(data, extType, rect)
	c := col.New(imgCols).Add(img)
	return row.New(heightMM).Add(c), true
}

func (tr *translator) inlineImage(n *dom.Node) (*props.RichImage, bool) {
	src := tr.selectedImageSource(n)
	if src == "" {
		return nil, false
	}
	return tr.inlineImageWithSource(n, src)
}

func (tr *translator) inlineImageWithSource(n *dom.Node, src string) (*props.RichImage, bool) {
	resolver := tr.imageResolver
	if resolver == nil {
		resolver = safeDefaultResolver
	}
	data, ext, err := resolver(src)
	if err != nil {
		tr.unsupported("img.src", err.Error())
		return nil, false
	}

	style := tr.imageStyle(n)
	dimensions := imageDimensions(n, style)
	intrinsicWidth, intrinsicHeight := 0.0, 0.0

	if ext == "svg" {
		pngBytes, w, h, rerr := rasteriseSVG(data, dimensions.width, dimensions.height)
		if rerr != nil {
			tr.unsupported("img.svg", rerr.Error())
			return nil, false
		}
		data = pngBytes
		ext = "png"
		intrinsicWidth = mmFromPx(w)
		intrinsicHeight = mmFromPx(h)
	} else {
		intrinsicWidth, intrinsicHeight = rasterImageSizeMM(data)
	}

	extType := extensionType(ext)
	if extType == "" {
		tr.unsupported("img.ext", "unsupported extension: "+ext)
		return nil, false
	}

	widthMM, heightMM := resolveImageDimensions(dimensions, intrinsicWidth, intrinsicHeight, 4)
	return &props.RichImage{
		Bytes:          data,
		Extension:      extType,
		Width:          widthMM,
		Height:         heightMM,
		Alt:            n.Attr("alt"),
		ObjectFit:      style.ObjectFit,
		ObjectPosition: style.ObjectPosition,
	}, true
}

func (tr *translator) inlinePicture(n *dom.Node) (*props.RichImage, bool) {
	img := pictureFallbackImage(n)
	if img == nil {
		return nil, false
	}
	src := pictureSelectedSource(n, strings.TrimSpace(img.Attr("src")))
	if src == "" {
		src = tr.selectedImageSource(img)
	}
	return tr.inlineImageWithSource(img, src)
}

func (tr *translator) pictureRow(n *dom.Node) []core.Row {
	img := pictureFallbackImage(n)
	if img == nil {
		return nil
	}
	src := pictureSelectedSource(n, strings.TrimSpace(img.Attr("src")))
	if src == "" {
		src = tr.selectedImageSource(img)
	}
	if r, ok := tr.imageRowWithSource(img, src); ok {
		return []core.Row{r}
	}
	return altRow(img)
}

func pictureFallbackImage(n *dom.Node) *dom.Node {
	for _, child := range n.Children() {
		if child.Tag() == "img" {
			return child
		}
	}
	return nil
}

func (tr *translator) selectedImageSource(n *dom.Node) string {
	return selectSrcsetCandidate(n.Attr("src"), n.Attr("srcset"))
}

func pictureSelectedSource(picture *dom.Node, fallback string) string {
	for _, child := range picture.Children() {
		if child.Tag() != "source" {
			continue
		}
		if src := selectSrcsetCandidate("", child.Attr("srcset")); src != "" {
			return src
		}
		if src := strings.TrimSpace(child.Attr("src")); src != "" {
			return src
		}
	}
	return fallback
}

type srcsetCandidate struct {
	src     string
	density float64
	width   float64
	order   int
}

func selectSrcsetCandidate(src, srcset string) string {
	fallback := strings.TrimSpace(src)
	candidates := parseSrcset(srcset)
	if len(candidates) == 0 {
		return fallback
	}
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if betterSrcsetCandidate(candidate, best) {
			best = candidate
		}
	}
	if best.src == "" {
		return fallback
	}
	return best.src
}

func parseSrcset(srcset string) []srcsetCandidate {
	var candidates []srcsetCandidate
	for i, raw := range splitSrcset(srcset) {
		fields := strings.Fields(raw)
		if len(fields) == 0 {
			continue
		}
		candidate := srcsetCandidate{src: fields[0], density: 1, order: i}
		for _, descriptor := range fields[1:] {
			if value, ok := strings.CutSuffix(descriptor, "x"); ok {
				if density, err := strconv.ParseFloat(value, 64); err == nil && density > 0 {
					candidate.density = density
				}
				continue
			}
			if value, ok := strings.CutSuffix(descriptor, "w"); ok {
				if width, err := strconv.ParseFloat(value, 64); err == nil && width > 0 {
					candidate.width = width
				}
			}
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func splitSrcset(srcset string) []string {
	var parts []string
	i := 0
	for i < len(srcset) {
		for i < len(srcset) && (srcset[i] == ',' || srcset[i] == ' ' || srcset[i] == '\t' || srcset[i] == '\n' || srcset[i] == '\r') {
			i++
		}
		start := i
		for i < len(srcset) && srcset[i] != ' ' && srcset[i] != '\t' && srcset[i] != '\n' && srcset[i] != '\r' {
			i++
		}
		for i < len(srcset) && srcset[i] != ',' {
			i++
		}
		if candidate := strings.TrimSpace(srcset[start:i]); candidate != "" {
			parts = append(parts, candidate)
		}
	}
	return parts
}

func betterSrcsetCandidate(candidate, best srcsetCandidate) bool {
	switch {
	case candidate.density != best.density:
		return candidate.density > best.density
	case candidate.width != best.width:
		return candidate.width > best.width
	default:
		return candidate.order < best.order
	}
}

func (tr *translator) svgRow(n *dom.Node) (core.Row, bool) {
	data, ok := svgElementBytes(n)
	if !ok {
		return nil, false
	}
	style := tr.imageStyle(n)
	dimensions := imageDimensions(n, style)
	pngBytes, widthPx, heightPx, err := rasteriseSVG(data, dimensions.width, dimensions.height)
	if err != nil {
		tr.unsupported("svg", err.Error())
		return nil, false
	}
	widthMM, heightMM := resolveImageDimensions(dimensions, mmFromPx(widthPx), mmFromPx(heightPx), 10)

	cellWidth := tr.contentWidthMM
	if cellWidth <= 0 {
		cellWidth = 170.0
	}
	gridSize := tr.gridSize
	if gridSize <= 0 {
		gridSize = defaultGridSize
	}
	rect := props.Rect{Percent: 100, Center: true}
	if style != nil {
		rect.ObjectFit = style.ObjectFit
		rect.ObjectPosition = style.ObjectPosition
	}
	img := imagecomp.NewFromBytes(pngBytes, extension.Png, rect)
	return row.New(heightMM).Add(col.New(imageCols(widthMM, cellWidth, gridSize)).Add(img)), true
}

func (tr *translator) inlineSVG(n *dom.Node) (*props.RichImage, bool) {
	data, ok := svgElementBytes(n)
	if !ok {
		return nil, false
	}
	style := tr.imageStyle(n)
	dimensions := imageDimensions(n, style)
	pngBytes, widthPx, heightPx, err := rasteriseSVG(data, dimensions.width, dimensions.height)
	if err != nil {
		tr.unsupported("svg", err.Error())
		return nil, false
	}
	widthMM, heightMM := resolveImageDimensions(dimensions, mmFromPx(widthPx), mmFromPx(heightPx), 4)
	return &props.RichImage{
		Bytes:          pngBytes,
		Extension:      extension.Png,
		Width:          widthMM,
		Height:         heightMM,
		ObjectFit:      style.ObjectFit,
		ObjectPosition: style.ObjectPosition,
	}, true
}

func svgElementBytes(n *dom.Node) ([]byte, bool) {
	if n == nil || n.Tag() != "svg" {
		return nil, false
	}
	var buf bytes.Buffer
	if err := html.Render(&buf, n.RawNode()); err != nil {
		return nil, false
	}
	data := buf.Bytes()
	data = bytes.ReplaceAll(data, []byte(" viewbox="), []byte(" viewBox="))
	return data, len(bytes.TrimSpace(data)) > 0
}

func (tr *translator) backgroundImage(style *css.ComputedStyle) *props.CellBackgroundImage {
	if style == nil {
		return nil
	}
	src := style.BackgroundImageURL
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}
	resolver := tr.imageResolver
	if resolver == nil {
		resolver = safeDefaultResolver
	}
	data, ext, err := resolver(src)
	if err != nil {
		tr.unsupported("background-image.src", err.Error())
		return nil
	}
	if ext == "svg" {
		pngBytes, _, _, rerr := rasteriseSVG(data, 0, 0)
		if rerr != nil {
			tr.unsupported("background-image.svg", rerr.Error())
			return nil
		}
		data = pngBytes
		ext = "png"
	}
	extType := extensionType(ext)
	if extType == "" {
		tr.unsupported("background-image.ext", "unsupported extension: "+ext)
		return nil
	}
	rect := props.Rect{Percent: 100, Center: true}
	rect.MakeValid()
	return &props.CellBackgroundImage{
		Bytes:     data,
		Extension: extType,
		Rect:      rect,
		Size:      style.BackgroundSize,
		Position:  style.BackgroundPosition,
		Repeat:    style.BackgroundRepeat,
	}
}

func (tr *translator) imageStyle(n *dom.Node) *css.ComputedStyle {
	return computeNodeStyleCtx(tr.sheet, n, tr.rootStyle, tr.availableContentWidth())
}

func (tr *translator) availableContentWidth() float64 {
	if tr.contentWidthMM > 0 {
		return tr.contentWidthMM
	}
	return 170.0
}

type imageDimensionStyle struct {
	width     float64
	height    float64
	minWidth  float64
	maxWidth  float64
	minHeight float64
	maxHeight float64
}

func imageDimensions(n *dom.Node, style *css.ComputedStyle) imageDimensionStyle {
	dimensions := imageDimensionStyle{
		width:  parseImageDimension(n.Attr("width")),
		height: parseImageDimension(n.Attr("height")),
	}
	if style != nil {
		if style.Width > 0 {
			dimensions.width = style.Width
		}
		if style.Height > 0 {
			dimensions.height = style.Height
		}
		dimensions.minWidth = style.MinWidth
		dimensions.maxWidth = style.MaxWidth
		dimensions.minHeight = style.MinHeight
		dimensions.maxHeight = style.MaxHeight
	}
	return dimensions
}

func rasterImageSizeMM(data []byte) (float64, float64) {
	cfg, _, err := goimage.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0
	}
	return css.ParseLength(strconv.Itoa(cfg.Width)+"px", 0), css.ParseLength(strconv.Itoa(cfg.Height)+"px", 0)
}

func resolveImageDimensions(dimensions imageDimensionStyle, intrinsicWidth, intrinsicHeight, fallbackSize float64) (float64, float64) {
	widthMM := dimensions.width
	heightMM := dimensions.height
	widthExplicit := widthMM > 0
	heightExplicit := heightMM > 0
	switch {
	case widthMM > 0 && heightMM > 0:
		// keep both explicit dimensions
	case widthMM > 0 && intrinsicWidth > 0 && intrinsicHeight > 0:
		heightMM = widthMM * intrinsicHeight / intrinsicWidth
	case heightMM > 0 && intrinsicWidth > 0 && intrinsicHeight > 0:
		widthMM = heightMM * intrinsicWidth / intrinsicHeight
	case widthMM > 0:
		heightMM = widthMM
	case heightMM > 0:
		widthMM = heightMM
	case intrinsicWidth > 0 && intrinsicHeight > 0:
		widthMM = intrinsicWidth
		heightMM = intrinsicHeight
	default:
		widthMM = fallbackSize
		heightMM = fallbackSize
	}
	return constrainImageDimensions(widthMM, heightMM, widthExplicit, heightExplicit, dimensions)
}

func constrainImageDimensions(widthMM, heightMM float64, widthExplicit, heightExplicit bool, dimensions imageDimensionStyle) (float64, float64) {
	aspect := 1.0
	if widthMM > 0 {
		aspect = heightMM / widthMM
	}
	if dimensions.minWidth > 0 && widthMM < dimensions.minWidth {
		widthMM = dimensions.minWidth
		if !heightExplicit {
			heightMM = widthMM * aspect
		}
	}
	if dimensions.maxWidth > 0 && widthMM > dimensions.maxWidth {
		widthMM = dimensions.maxWidth
		if !heightExplicit {
			heightMM = widthMM * aspect
		}
	}
	if dimensions.minHeight > 0 && heightMM < dimensions.minHeight {
		heightMM = dimensions.minHeight
		if !widthExplicit && aspect > 0 {
			widthMM = heightMM / aspect
		}
	}
	if dimensions.maxHeight > 0 && heightMM > dimensions.maxHeight {
		heightMM = dimensions.maxHeight
		if !widthExplicit && aspect > 0 {
			widthMM = heightMM / aspect
		}
	}
	return widthMM, heightMM
}

// unsupported reports a rendering issue back through the optional handler.
func (tr *translator) unsupported(kind, msg string) {
	if tr.unsupportedHandler != nil {
		tr.unsupportedHandler(kind, msg)
	}
}

// altRow renders the <img>'s alt text as a paragraph row (fallback path).
func altRow(n *dom.Node) []core.Row {
	alt := strings.TrimSpace(n.Attr("alt"))
	if alt == "" {
		return nil
	}
	rt := richtext.New([]props.RichRun{{Text: alt}})
	c := col.New().Add(rt)
	return []core.Row{row.New().Add(c)}
}

func imageCols(widthMM, cellWidth float64, gridSize int) int {
	if gridSize <= 0 {
		gridSize = defaultGridSize
	}
	if widthMM <= 0 || cellWidth <= 0 {
		return gridSize
	}
	mmPerCol := cellWidth / float64(gridSize)
	if mmPerCol <= 0 {
		return gridSize
	}
	cols := int(widthMM/mmPerCol + 0.5)
	if cols < 1 {
		return 1
	}
	if cols > gridSize {
		return gridSize
	}
	return cols
}

// parseImageDimension parses a CSS length string (px/pt/mm/cm) into mm.
// Returns 0 for empty or unparseable input.
func parseImageDimension(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// If the value is unit-less (e.g. width="20"), treat it as px.
	hasUnit := false
	for _, u := range []string{"px", "pt", "mm", "cm"} {
		if strings.HasSuffix(s, u) {
			hasUnit = true
			break
		}
	}
	if !hasUnit {
		s += "px"
	}
	return css.ParseLength(s, 0)
}

// rasteriseSVG converts SVG bytes into PNG bytes at the requested mm dimensions
// (using dpiForRaster). Returns the PNG, width-px, height-px, and any error.
// When widthMM/heightMM are 0 it uses the SVG's intrinsic ViewBox to pick a size.
func rasteriseSVG(svgBytes []byte, widthMM, heightMM float64) ([]byte, int, int, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgBytes), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("html: svg parse: %w", err)
	}
	pxW, pxH := svgTargetPixels(icon, widthMM, heightMM)
	if pxW <= 0 || pxH <= 0 {
		return nil, 0, 0, fmt.Errorf("html: svg has zero dimensions")
	}
	icon.SetTarget(0, 0, float64(pxW), float64(pxH))
	rgba := goimage.NewRGBA(goimage.Rect(0, 0, pxW, pxH))
	scanner := rasterx.NewScannerGV(pxW, pxH, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(pxW, pxH, scanner)
	icon.Draw(dasher, 1.0)

	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, 0, 0, fmt.Errorf("html: png encode: %w", err)
	}
	return buf.Bytes(), pxW, pxH, nil
}

func svgTargetPixels(icon *oksvg.SvgIcon, widthMM, heightMM float64) (int, int) {
	vbW := icon.ViewBox.W
	vbH := icon.ViewBox.H
	switch {
	case widthMM > 0 && heightMM > 0:
		return pxFromMM(widthMM), pxFromMM(heightMM)
	case widthMM > 0:
		if vbW > 0 && vbH > 0 {
			pxW := pxFromMM(widthMM)
			pxH := int(float64(pxW) * vbH / vbW)
			return pxW, pxH
		}
		return pxFromMM(widthMM), pxFromMM(widthMM)
	case heightMM > 0:
		if vbW > 0 && vbH > 0 {
			pxH := pxFromMM(heightMM)
			pxW := int(float64(pxH) * vbW / vbH)
			return pxW, pxH
		}
		return pxFromMM(heightMM), pxFromMM(heightMM)
	default:
		if vbW > 0 && vbH > 0 {
			return int(vbW), int(vbH)
		}
		return 32, 32
	}
}

// pxFromMM converts millimetres to pixels at dpiForRaster.
func pxFromMM(mm float64) int {
	px := int(mm / 25.4 * dpiForRaster)
	if px < 1 {
		px = 1
	}
	return px
}

// mmFromPx converts pixels back to millimetres at dpiForRaster.
func mmFromPx(px int) float64 {
	return float64(px) / dpiForRaster * 25.4
}

func extensionType(ext string) extension.Type {
	switch strings.ToLower(ext) {
	case "png":
		return extension.Png
	case "jpg", "jpeg":
		return extension.Jpg
	default:
		return ""
	}
}
