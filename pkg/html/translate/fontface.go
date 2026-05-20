package translate

import (
	"strings"

	"github.com/aymerick/douceur/css"
	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
)

// loadedFont is a font that's been resolved via the stylesheet resolver
// and is ready to be registered with the gofpdf provider via LateFontProvider
// at render time.
type loadedFont struct {
	family string
	bytes  []byte
}

// fontRegistration is a zero-height component that registers a font with
// the provider on first Render. Subsequent renders are no-ops (registration
// is idempotent within a single document; gofpdf overwrites silently).
type fontRegistration struct {
	font loadedFont
	done bool
}

func (f *fontRegistration) SetConfig(*entity.Config) {}

func (f *fontRegistration) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{
		Type:    "font_registration",
		Details: map[string]any{"family": f.font.family},
	})
}

func (f *fontRegistration) GetHeight(_ core.Provider, _ *entity.Cell) float64 {
	return 0
}

func (f *fontRegistration) Render(provider core.Provider, _ *entity.Cell) {
	if f.done {
		return
	}
	if lfp, ok := provider.(core.LateFontProvider); ok {
		lfp.RegisterFont(f.font.family, fontstyle.Normal, f.font.bytes)
		f.done = true
	}
}

// registerFontFaces is called by Translate after parseStylesheet to load
// @font-face URLs and produce hidden registration rows. The rows are
// prepended to the final output so the font is registered before any
// subsequent text that uses font-family: "MyFont" renders.
func (tr *translator) registerFontFaces(resolver StylesheetResolver) {
	if tr.sheet == nil || len(tr.sheet.fontFaces) == 0 {
		return
	}
	if resolver == nil {
		resolver = safeDefaultStylesheetResolver
	}
	for _, face := range tr.sheet.fontFaces {
		data, ok := safeLoadStylesheet(resolver, face.srcURL)
		if !ok {
			if tr.unsupportedHandler != nil {
				tr.unsupportedHandler("font-face.skipped", face.srcURL)
			}
			continue
		}
		tr.loadedFonts = append(tr.loadedFonts, loadedFont{
			family: face.family,
			bytes:  data,
		})
	}
}

// fontRegistrationRows returns the zero-height rows that register all loaded
// fonts. Prepended to the output by Translate so registration occurs before
// any consuming text renders.
func (tr *translator) fontRegistrationRows() []core.Row {
	out := make([]core.Row, 0, len(tr.loadedFonts))
	for i := range tr.loadedFonts {
		reg := &fontRegistration{font: tr.loadedFonts[i]}
		out = append(out, row.New(0).Add(col.New().Add(reg)))
	}
	return out
}

// fontFaceRule is the extracted shape of a CSS @font-face rule.
// Douceur surfaces @font-face as an AtRule with Name == "@font-face" and a
// declaration list. The src value is a raw string like:
//
//	url("./foo.ttf") format("truetype"), local("Foo")
//
// We extract the first valid url() with format="truetype" or "opentype" —
// woff/woff2 are skipped because the underlying gofpdf fork cannot decode
// them. local() entries are also skipped (no system-font lookup in PDF).
type fontFaceRule struct {
	family string
	srcURL string
}

// extractFontFace pulls family and src URL from a parsed @font-face rule.
// Returns ok=false on missing/malformed family or src.
func extractFontFace(rule *css.Rule) (fontFaceRule, bool) {
	out := fontFaceRule{}
	for _, d := range rule.Declarations {
		switch strings.ToLower(d.Property) {
		case "font-family":
			// Lower-case the family name so case-insensitive matching against
			// CSS `font-family: "MyFont"` declarations works predictably.
			out.family = strings.ToLower(strings.Trim(strings.TrimSpace(d.Value), `"'`))
		case "src":
			if u, ok := parseFontFaceSrc(d.Value); ok {
				out.srcURL = u
			}
		}
	}
	if out.family == "" || out.srcURL == "" {
		return out, false
	}
	return out, true
}

// parseFontFaceSrc parses a src: declaration value and returns the FIRST
// url() reference whose adjacent format() is truetype or opentype. The
// grammar in CSS is comma-separated descriptors; we tokenise leniently.
//
// Examples accepted:
//
//	url("./foo.ttf") format("truetype")
//	local("Foo"), url("foo.otf") format("opentype")
//	url(foo.ttf) format(truetype)
//
// Examples rejected (return ok=false):
//
//	url("./foo.woff") format("woff")
//	local("Foo")
func parseFontFaceSrc(value string) (string, bool) {
	for _, descriptor := range strings.Split(value, ",") {
		d := strings.TrimSpace(descriptor)
		if !strings.HasPrefix(d, "url(") {
			continue
		}
		closeIdx := strings.IndexByte(d, ')')
		if closeIdx < 0 {
			continue
		}
		url := strings.Trim(strings.TrimSpace(d[4:closeIdx]), `"'`)
		rest := strings.TrimSpace(d[closeIdx+1:])
		// Format check: must contain format(truetype) or format(opentype).
		// If no format() at all, assume truetype (browser default).
		if rest == "" {
			return url, url != ""
		}
		lr := strings.ToLower(rest)
		if strings.Contains(lr, "format(\"truetype\")") ||
			strings.Contains(lr, "format('truetype')") ||
			strings.Contains(lr, "format(truetype)") ||
			strings.Contains(lr, "format(\"opentype\")") ||
			strings.Contains(lr, "format('opentype')") ||
			strings.Contains(lr, "format(opentype)") {
			return url, url != ""
		}
	}
	return "", false
}
