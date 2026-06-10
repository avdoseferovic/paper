package translate

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	goimage "image"
	_ "image/jpeg"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/avdoseferovic/paper/internal/htmllimits"
	svgraster "github.com/avdoseferovic/paper/internal/svg"
	"github.com/avdoseferovic/paper/pkg/components/col"
	imagecomp "github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
	"golang.org/x/net/html"
)

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

var (
	errDataURIInvalid      = errors.New("html: invalid data URI")
	errAbsolutePathRefused = errors.New("html: absolute path refused outside base dir")
	errPathEscapesBaseDir  = errors.New("html: path escapes base dir")
	errBaseDirEmpty        = errors.New("html: base dir is empty; refusing all local reads")
)

// safeDefaultResolver only accepts data: URIs. It refuses any other src to
// prevent path-traversal attacks on user-controlled HTML.
func safeDefaultResolver(src string) ([]byte, string, error) {
	return safeDefaultResolverWithLimits(src, htmllimits.Default())
}

func safeDefaultResolverWithLimits(src string, limits htmllimits.Limits) ([]byte, string, error) {
	if strings.HasPrefix(src, "data:") {
		return decodeDataURIWithLimits(src, limits)
	}
	return nil, "", ErrImageResolverRefused
}

// decodeDataURI parses a data: URI of the form data:<mime>;base64,<payload>.
// Returns the decoded bytes and an extension hint inferred from the MIME type.
func decodeDataURI(uri string) ([]byte, string, error) {
	return decodeDataURIWithLimits(uri, htmllimits.Default())
}

func decodeDataURIWithLimits(uri string, limits htmllimits.Limits) ([]byte, string, error) {
	// data:<mediatype>;base64,<payload>
	prefix := strings.TrimPrefix(uri, "data:")
	header, payload, ok := strings.Cut(prefix, ",")
	if !ok {
		return nil, "", errDataURIInvalid
	}
	mediaType := header
	isB64 := false
	if before, after, ok := strings.Cut(header, ";"); ok {
		mediaType = before
		if strings.Contains(after, "base64") {
			isB64 = true
		}
	}
	var data []byte
	if isB64 {
		estimatedBytes := int64(base64.StdEncoding.DecodedLen(len(payload)))
		if htmllimits.Int64Exceeded(limits.MaxImageBytes, estimatedBytes) {
			return nil, "", fmt.Errorf(
				"%w: data URI payload decodes to at most %d bytes; limit %d",
				htmllimits.ErrImageTooLarge,
				estimatedBytes,
				limits.MaxImageBytes,
			)
		}
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, "", fmt.Errorf("html: data URI base64 decode: %w", err)
		}
		data = decoded
	} else {
		if htmllimits.Int64Exceeded(limits.MaxImageBytes, int64(len(payload))) {
			return nil, "", fmt.Errorf("%w: data URI payload is %d bytes; limit %d", htmllimits.ErrImageTooLarge, len(payload), limits.MaxImageBytes)
		}
		data = []byte(payload)
	}
	ext := imageExtPNG
	switch mediaType {
	case "image/png":
		ext = imageExtPNG
	case "image/jpeg", "image/jpg":
		ext = imageExtJPG
	case "image/svg+xml":
		ext = imageExtSVG
	}
	return data, ext, nil
}

// baseDirResolver returns a resolver that only loads files inside dir,
// rejecting any path that would escape via "../" or absolute prefix.
func baseDirResolver(dir string) ImageResolver {
	return baseDirResolverWithLimits(dir, htmllimits.Default())
}

func baseDirResolverWithLimits(dir string, limits htmllimits.Limits) ImageResolver {
	return func(src string) ([]byte, string, error) {
		if strings.HasPrefix(src, "data:") {
			return decodeDataURIWithLimits(src, limits)
		}
		data, err := readFileInRoot(dir, src)
		if err != nil {
			return nil, "", err
		}
		return data, extFromFilename(src), nil
	}
}

func (tr *translator) resolveImage(src string) ([]byte, string, error) {
	if tr.imageResolver != nil {
		return tr.imageResolver(src)
	}
	if tr.imageBaseDir != "" {
		return baseDirResolverWithLimits(tr.imageBaseDir, tr.limits)(src)
	}
	return safeDefaultResolverWithLimits(src, tr.limits)
}

func extFromFilename(name string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	switch ext {
	case "jpeg":
		return imageExtJPG
	default:
		return ext
	}
}

// imageRow builds a block-level row for <img>. Returns the row and ok=true on
// success; ok=false signals the caller to fall back to alt text.
func (tr *translator) imageRowWithStyle(n *dom.Node, style *css.ComputedStyle) (core.Row, bool) {
	src := tr.selectedImageSource(n)
	if src == "" {
		return nil, false
	}
	return tr.imageRowWithSourceAndStyle(n, src, style)
}

func (tr *translator) imageRowWithSourceAndStyle(n *dom.Node, src string, style *css.ComputedStyle) (core.Row, bool) {
	data, ext, err := tr.resolveImage(src)
	if err != nil {
		tr.unsupported("img.src", err.Error())
		if errors.Is(err, htmllimits.ErrImageTooLarge) {
			tr.err = err
		}
		return nil, false
	}

	if style == nil {
		style = tr.imageStyle(n)
	}
	dimensions := imageDimensions(n, style)
	intrinsicWidth, intrinsicHeight := 0.0, 0.0

	data, ext, intrinsicWidth, intrinsicHeight, ok := tr.prepareImageData(data, ext, dimensions, "img")
	if !ok {
		return nil, false
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
		cellWidth = defaultContentWidthMM
	}
	gridSize := tr.gridSize
	if gridSize <= 0 {
		gridSize = defaultGridSize
	}
	imgCols := imageCols(widthMM, cellWidth, gridSize)
	if isVisibilityHidden(style) {
		return row.New(heightMM).Add(col.New(imgCols)), true
	}

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
	style := tr.imageStyle(n)
	return tr.richImageFromSource(src, style, imageDimensions(n, style), n.Attr("alt"), tagImg)
}

func (tr *translator) generatedContentImage(src string, style *css.ComputedStyle) (*props.RichImage, bool) {
	return tr.richImageFromSource(src, style, imageDimensionsFromStyle(style), "", "content.url")
}

func (tr *translator) richImageFromSource(
	src string,
	style *css.ComputedStyle,
	dimensions imageDimensionStyle,
	alt string,
	unsupportedPrefix string,
) (*props.RichImage, bool) {
	data, ext, err := tr.resolveImage(src)
	if err != nil {
		tr.unsupported(unsupportedPrefix+".src", err.Error())
		if errors.Is(err, htmllimits.ErrImageTooLarge) {
			tr.err = err
		}
		return nil, false
	}

	data, ext, intrinsicWidth, intrinsicHeight, ok := tr.prepareImageData(data, ext, dimensions, unsupportedPrefix)
	if !ok {
		return nil, false
	}

	extType := extensionType(ext)
	if extType == "" {
		tr.unsupported(unsupportedPrefix+".ext", "unsupported extension: "+ext)
		return nil, false
	}

	widthMM, heightMM := resolveImageDimensions(dimensions, intrinsicWidth, intrinsicHeight, 4)
	objectFit, objectPosition := "", ""
	if style != nil {
		objectFit = style.ObjectFit
		objectPosition = style.ObjectPosition
	}
	return &props.RichImage{
		Bytes:          data,
		Extension:      extType,
		Width:          widthMM,
		Height:         heightMM,
		Alt:            alt,
		ObjectFit:      objectFit,
		ObjectPosition: objectPosition,
	}, true
}

func (tr *translator) inlinePicture(n *dom.Node) (*props.RichImage, bool) {
	img := pictureFallbackImage(n)
	if img == nil {
		return nil, false
	}
	src := tr.pictureSelectedSource(n, img)
	if src == "" {
		src = tr.selectedImageSource(img)
	}
	return tr.inlineImageWithSource(img, src)
}

func (tr *translator) pictureRowWithStyle(n *dom.Node, style *css.ComputedStyle) []core.Row {
	img := pictureFallbackImage(n)
	if img == nil {
		return nil
	}
	if style == nil {
		style = tr.rootStyle
	}
	src := tr.pictureSelectedSource(n, img)
	if src == "" {
		src = tr.selectedImageSource(img)
	}
	imgStyle := tr.imageStyleWithParent(img, style)
	if r, ok := tr.imageRowWithSourceAndStyle(img, src, imgStyle); ok {
		return []core.Row{r}
	}
	return altRowStyled(img, imgStyle)
}

func pictureFallbackImage(n *dom.Node) *dom.Node {
	for _, child := range n.Children() {
		if child.Tag() == tagImg {
			return child
		}
	}
	return nil
}

func (tr *translator) selectedImageSource(n *dom.Node) string {
	return selectSrcsetCandidate(n.Attr("src"), n.Attr("srcset"), n.Attr("sizes"), tr.availableContentWidth())
}

func (tr *translator) pictureSelectedSource(picture, fallbackImg *dom.Node) string {
	fallback := strings.TrimSpace(fallbackImg.Attr("src"))
	fallbackSizes := fallbackImg.Attr("sizes")
	for _, child := range picture.Children() {
		if child.Tag() != "source" {
			continue
		}
		if !pictureSourceApplies(child, tr.availableContentWidth()) {
			continue
		}
		sizes := child.Attr("sizes")
		if strings.TrimSpace(sizes) == "" {
			sizes = fallbackSizes
		}
		if src := selectSrcsetCandidate("", child.Attr("srcset"), sizes, tr.availableContentWidth()); src != "" {
			return src
		}
		if src := strings.TrimSpace(child.Attr("src")); src != "" {
			return src
		}
	}
	return fallback
}

func pictureSourceApplies(source *dom.Node, contentWidthMM float64) bool {
	return pictureSourceTypeSupported(source.Attr("type")) && pictureSourceMediaApplies(source.Attr("media"), contentWidthMM)
}

func pictureSourceTypeSupported(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return true
	}
	if mediaType, _, ok := strings.Cut(value, ";"); ok {
		value = strings.TrimSpace(mediaType)
	}
	switch value {
	case "image/png", "image/jpeg", "image/jpg", "image/svg+xml":
		return true
	default:
		return false
	}
}

func pictureSourceMediaApplies(value string, contentWidthMM float64) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	return mediaAppliesToPrintAtWidth(value, contentWidthMM)
}

type srcsetCandidate struct {
	src     string
	density float64
	width   float64
	order   int
}

func selectSrcsetCandidate(src, srcset, sizes string, contentWidthMM float64) string {
	fallback := strings.TrimSpace(src)
	candidates := parseSrcset(srcset)
	if len(candidates) == 0 {
		return fallback
	}
	if slotWidthPx := sourceSizePx(sizes, contentWidthMM); slotWidthPx > 0 {
		if selected := selectWidthDescriptorCandidate(candidates, slotWidthPx); selected != "" {
			return selected
		}
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

func selectWidthDescriptorCandidate(candidates []srcsetCandidate, slotWidthPx float64) string {
	var best *srcsetCandidate
	for i := range candidates {
		candidate := &candidates[i]
		if candidate.width <= 0 || candidate.src == "" {
			continue
		}
		if candidate.width >= slotWidthPx {
			if bestWidthCandidate(candidate, best, slotWidthPx) {
				best = candidate
			}
			continue
		}
		if best == nil || best.width < slotWidthPx && candidate.width > best.width {
			best = candidate
		}
	}
	if best == nil {
		return ""
	}
	return best.src
}

func bestWidthCandidate(candidate *srcsetCandidate, best *srcsetCandidate, slotWidthPx float64) bool {
	return best == nil ||
		best.width < slotWidthPx ||
		candidate.width < best.width ||
		candidate.width == best.width && candidate.order < best.order
}

func sourceSizePx(sizes string, contentWidthMM float64) float64 {
	sizeMM := sourceSizeMM(sizes, contentWidthMM)
	if sizeMM <= 0 {
		return 0
	}
	return sizeMM / 0.264583
}

func sourceSizeMM(sizes string, contentWidthMM float64) float64 {
	if contentWidthMM <= 0 {
		contentWidthMM = defaultContentWidthMM
	}
	for _, raw := range splitSizesList(sizes) {
		size, ok := parseSourceSize(raw, contentWidthMM)
		if ok && size > 0 {
			return size
		}
	}
	return 0
}

func splitSizesList(sizes string) []string {
	var parts []string
	start := 0
	depth := 0
	var quote rune
	for i, r := range sizes {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		default:
			switch r {
			case '"', '\'':
				quote = r
			case '(':
				depth++
			case ')':
				if depth > 0 {
					depth--
				}
			case ',':
				if depth == 0 {
					parts = append(parts, strings.TrimSpace(sizes[start:i]))
					start = i + 1
				}
			}
		}
	}
	if tail := strings.TrimSpace(sizes[start:]); tail != "" {
		parts = append(parts, tail)
	}
	return parts
}

func parseSourceSize(value string, contentWidthMM float64) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	lengthStart := lastSourceSizeLengthStart(value)
	if lengthStart < 0 {
		return 0, false
	}
	media := strings.TrimSpace(value[:lengthStart])
	if media != "" && !mediaAppliesToPrintAtWidth(media, contentWidthMM) {
		return 0, false
	}
	length := strings.TrimSpace(value[lengthStart:])
	return parseSourceSizeLength(length, contentWidthMM)
}

func lastSourceSizeLengthStart(value string) int {
	depth := 0
	var quote rune
	for i := len(value) - 1; i >= 0; i-- {
		r := rune(value[i])
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		default:
			switch r {
			case '"', '\'':
				quote = r
			case ')':
				depth++
			case '(':
				if depth > 0 {
					depth--
				}
			case ' ', '\t', '\r', '\n', '\f':
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return 0
}

func parseSourceSizeLength(length string, contentWidthMM float64) (float64, bool) {
	length = strings.TrimSpace(length)
	if length == "" || strings.EqualFold(length, cssValueAuto) {
		return 0, false
	}
	if strings.HasSuffix(strings.ToLower(length), "vw") {
		v, err := strconv.ParseFloat(strings.TrimSpace(length[:len(length)-2]), 64)
		if err != nil {
			return 0, false
		}
		return contentWidthMM * v / 100, true
	}
	if strings.Contains(length, "%") || strings.Contains(length, "calc(") {
		return css.ParseLengthCtx(length, 0, contentWidthMM), true
	}
	return css.ParseLength(length, 0), true
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
				density, err := strconv.ParseFloat(value, 64)
				if err == nil && density > 0 {
					candidate.density = density
				}
				continue
			}
			if value, ok := strings.CutSuffix(descriptor, "w"); ok {
				width, err := strconv.ParseFloat(value, 64)
				if err == nil && width > 0 {
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

func (tr *translator) svgRowWithStyle(n *dom.Node, style *css.ComputedStyle) (core.Row, bool) {
	data, ok := svgElementBytes(n)
	if !ok {
		return nil, false
	}
	if style == nil {
		style = tr.imageStyle(n)
	}
	dimensions := imageDimensions(n, style)
	pngBytes, widthPx, heightPx, err := svgraster.RasterizeWithLimit(data, dimensions.width, dimensions.height, tr.limits.MaxSVGPixels)
	if err != nil {
		tr.unsupported(tagSVG, err.Error())
		if errors.Is(err, htmllimits.ErrSVGTooLarge) {
			tr.err = err
		}
		return nil, false
	}
	widthMM, heightMM := resolveImageDimensions(dimensions, svgraster.MMFromPx(widthPx), svgraster.MMFromPx(heightPx), 10)

	cellWidth := tr.contentWidthMM
	if cellWidth <= 0 {
		cellWidth = defaultContentWidthMM
	}
	gridSize := tr.gridSize
	if gridSize <= 0 {
		gridSize = defaultGridSize
	}
	imgCols := imageCols(widthMM, cellWidth, gridSize)
	if isVisibilityHidden(style) {
		return row.New(heightMM).Add(col.New(imgCols)), true
	}
	rect := props.Rect{Percent: 100, Center: true}
	if style != nil {
		rect.ObjectFit = style.ObjectFit
		rect.ObjectPosition = style.ObjectPosition
	}
	img := imagecomp.NewFromBytes(pngBytes, extension.Png, rect)
	return row.New(heightMM).Add(col.New(imgCols).Add(img)), true
}

func (tr *translator) inlineSVG(n *dom.Node) (*props.RichImage, bool) {
	data, ok := svgElementBytes(n)
	if !ok {
		return nil, false
	}
	style := tr.imageStyle(n)
	dimensions := imageDimensions(n, style)
	pngBytes, widthPx, heightPx, err := svgraster.RasterizeWithLimit(data, dimensions.width, dimensions.height, tr.limits.MaxSVGPixels)
	if err != nil {
		tr.unsupported(tagSVG, err.Error())
		if errors.Is(err, htmllimits.ErrSVGTooLarge) {
			tr.err = err
		}
		return nil, false
	}
	widthMM, heightMM := resolveImageDimensions(dimensions, svgraster.MMFromPx(widthPx), svgraster.MMFromPx(heightPx), 4)
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
	if n == nil || n.Tag() != tagSVG {
		return nil, false
	}
	var buf bytes.Buffer
	err := html.Render(&buf, n.RawNode())
	if err != nil {
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
	data, ext, err := tr.resolveImage(src)
	if err != nil {
		tr.unsupported("background-image.src", err.Error())
		if errors.Is(err, htmllimits.ErrImageTooLarge) {
			tr.err = err
		}
		return nil
	}
	if ext == imageExtSVG {
		pngBytes, _, _, rerr := svgraster.RasterizeWithLimit(data, 0, 0, tr.limits.MaxSVGPixels)
		if rerr != nil {
			tr.unsupported("background-image.svg", rerr.Error())
			if errors.Is(rerr, htmllimits.ErrSVGTooLarge) {
				tr.err = rerr
			}
			return nil
		}
		data = pngBytes
		ext = imageExtPNG
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
	return tr.imageStyleWithParent(n, tr.rootStyle)
}

func (tr *translator) imageStyleWithParent(n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle {
	return computeNodeStyleCtx(tr.sheet, n, parent, tr.availableContentWidth())
}

func (tr *translator) availableContentWidth() float64 {
	if tr.contentWidthMM > 0 {
		return tr.contentWidthMM
	}
	return defaultContentWidthMM
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
	return applyImageDimensionStyle(dimensions, style)
}

func imageDimensionsFromStyle(style *css.ComputedStyle) imageDimensionStyle {
	return applyImageDimensionStyle(imageDimensionStyle{}, style)
}

func (tr *translator) prepareImageData(
	data []byte,
	ext string,
	dimensions imageDimensionStyle,
	unsupportedPrefix string,
) ([]byte, string, float64, float64, bool) {
	if ext == imageExtSVG {
		pngBytes, w, h, err := svgraster.RasterizeWithLimit(
			data,
			dimensions.width,
			dimensions.height,
			tr.limits.MaxSVGPixels,
		)
		if err != nil {
			tr.unsupported(unsupportedPrefix+".svg", err.Error())
			if errors.Is(err, htmllimits.ErrSVGTooLarge) {
				tr.err = err
			}
			return nil, "", 0, 0, false
		}
		return pngBytes, imageExtPNG, svgraster.MMFromPx(w), svgraster.MMFromPx(h), true
	}

	intrinsicWidth, intrinsicHeight, err := tr.rasterImageSizeMM(data)
	if err != nil {
		tr.unsupported(unsupportedPrefix+".size", err.Error())
		tr.err = err
		return nil, "", 0, 0, false
	}
	return data, ext, intrinsicWidth, intrinsicHeight, true
}

func applyImageDimensionStyle(dimensions imageDimensionStyle, style *css.ComputedStyle) imageDimensionStyle {
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

func (tr *translator) rasterImageSizeMM(data []byte) (float64, float64, error) {
	cfg, _, err := goimage.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// Malformed images have no intrinsic size here; the renderer reports
		// decode failures later if the caller still embeds the bytes.
		return unknownRasterImageSize()
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, nil
	}
	pixels := int64(cfg.Width) * int64(cfg.Height)
	if htmllimits.Int64Exceeded(tr.limits.MaxImagePixels, pixels) {
		return 0, 0, fmt.Errorf(
			"%w: %dx%d image has %d pixels; limit %d",
			htmllimits.ErrImageTooLarge,
			cfg.Width,
			cfg.Height,
			pixels,
			tr.limits.MaxImagePixels,
		)
	}
	return css.ParseLength(strconv.Itoa(cfg.Width)+"px", 0), css.ParseLength(strconv.Itoa(cfg.Height)+"px", 0), nil
}

func unknownRasterImageSize() (float64, float64, error) {
	return 0, 0, nil
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

func constrainImageDimensions(
	widthMM, heightMM float64,
	widthExplicit, heightExplicit bool,
	dimensions imageDimensionStyle,
) (float64, float64) {
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
func altRowStyled(n *dom.Node, style *css.ComputedStyle) []core.Row {
	alt := strings.TrimSpace(n.Attr("alt"))
	if alt == "" {
		return nil
	}
	run := props.RichRun{Text: alt}
	applyInlineStyleToRun(style, &run)
	rt := richtext.New([]props.RichRun{run})
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

func extensionType(ext string) extension.Type {
	switch strings.ToLower(ext) {
	case imageExtPNG:
		return extension.Png
	case imageExtJPG, "jpeg":
		return extension.Jpg
	default:
		return ""
	}
}
