package translate

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	goimage "image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/components/col"
	imagecomp "github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/richtext"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// dpiForRaster is the DPI used when rasterising SVG to PNG for PDF embedding.
const dpiForRaster = 150.0

// ImageResolver loads image bytes for a given <img src="…"> value. It returns
// the raw bytes and a hint extension ("png", "jpg", "svg", etc.). When err is
// non-nil the caller falls back to the <img>'s alt text.
type ImageResolver func(src string) (data []byte, ext string, err error)

// ErrImageResolverRefused is returned by the default resolver when asked to load
// a non-data: URI without an explicit WithImageBaseDir or WithImageResolver.
var ErrImageResolverRefused = errors.New("html: default resolver refuses local file reads; configure WithImageBaseDir or WithImageResolver")

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
	src := strings.TrimSpace(n.Attr("src"))
	if src == "" {
		return nil, false
	}
	resolver := tr.imageResolver
	if resolver == nil {
		resolver = safeDefaultResolver
	}
	data, ext, err := resolver(src)
	if err != nil {
		tr.unsupported("img.src", err.Error())
		return nil, false
	}

	widthMM := parseImageDimension(n.Attr("width"))
	heightMM := parseImageDimension(n.Attr("height"))

	if ext == "svg" {
		pngBytes, w, h, rerr := rasteriseSVG(data, widthMM, heightMM)
		if rerr != nil {
			tr.unsupported("img.svg", rerr.Error())
			return nil, false
		}
		data = pngBytes
		ext = "png"
		if widthMM == 0 {
			widthMM = mmFromPx(w)
		}
		if heightMM == 0 {
			heightMM = mmFromPx(h)
		}
	}

	extType := extensionType(ext)
	if extType == "" {
		tr.unsupported("img.ext", "unsupported extension: "+ext)
		return nil, false
	}

	if heightMM <= 0 {
		heightMM = widthMM
	}
	if heightMM <= 0 {
		heightMM = 10 // safe default
	}

	img := imagecomp.NewFromBytes(data, extType, props.Rect{Percent: 100, Center: true})
	c := col.New().Add(img)
	return row.New(heightMM).Add(c), true
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
