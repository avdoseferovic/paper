package translate

import (
	"errors"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// lateFontRecordingProvider extends cursorProvider with core.LateFontProvider.
type lateFontRecordingProvider struct {
	cursorProvider
	registered []string
}

func (p *lateFontRecordingProvider) RegisterFont(family string, _ fontstyle.Type, _ []byte) {
	p.registered = append(p.registered, family)
}

func TestFontRegistration_GetHeightIsZero(t *testing.T) {
	t.Parallel()
	f := &fontRegistration{font: loadedFont{family: "x"}}
	assert.Equal(t, 0.0, f.GetHeight(nil, nil))
}

func TestFontRegistration_SetConfigIsNoOp(t *testing.T) {
	t.Parallel()
	f := &fontRegistration{}
	f.SetConfig(&entity.Config{}) // must not panic
	f.SetConfig(nil)
}

func TestFontRegistration_GetStructure_IncludesFamily(t *testing.T) {
	t.Parallel()
	f := &fontRegistration{font: loadedFont{family: "myfont"}}
	str := f.GetStructure()
	assert.Equal(t, "font_registration", str.GetData().Type)
	assert.Equal(t, "myfont", str.GetData().Details["family"])
}

func TestFontRegistration_Render_RegistersOnce(t *testing.T) {
	t.Parallel()
	f := &fontRegistration{font: loadedFont{family: "myfont", bytes: []byte{1, 2}}}
	p := &lateFontRecordingProvider{}

	f.Render(p, nil)
	f.Render(p, nil) // second render must be a no-op

	assert.Equal(t, []string{"myfont"}, p.registered)
	assert.True(t, f.done)
}

func TestFontRegistration_Render_SkipsPlainProvider(t *testing.T) {
	t.Parallel()
	f := &fontRegistration{font: loadedFont{family: "myfont"}}

	f.Render(&cursorProvider{}, nil)

	assert.False(t, f.done, "non-LateFontProvider must not mark registration done")
}

func TestRegisterFontFaces_NilSheetIsNoOp(t *testing.T) {
	t.Parallel()
	tr := &translator{}
	tr.registerFontFaces(nil)
	assert.Empty(t, tr.loadedFonts)
}

func TestRegisterFontFaces_EmptyFontFacesIsNoOp(t *testing.T) {
	t.Parallel()
	tr := &translator{sheet: &stylesheet{}}
	tr.registerFontFaces(nil)
	assert.Empty(t, tr.loadedFonts)
}

func TestRegisterFontFaces_ResolverFailureReportsSkip(t *testing.T) {
	t.Parallel()
	var got []string
	tr := &translator{
		sheet: &stylesheet{fontFaces: []fontFaceRule{{family: "myfont", srcURL: "missing.ttf"}}},
		unsupportedHandler: func(thing, value string) {
			got = append(got, thing+"="+value)
		},
	}

	// nil resolver falls back to the safe default, which refuses local files.
	tr.registerFontFaces(nil)

	assert.Empty(t, tr.loadedFonts)
	assert.Contains(t, got, "font-face.skipped=missing.ttf")
}

func TestRegisterFontFaces_ResolverFailureWithoutHandlerDoesNotPanic(t *testing.T) {
	t.Parallel()
	tr := &translator{
		sheet: &stylesheet{fontFaces: []fontFaceRule{{family: "myfont", srcURL: "missing.ttf"}}},
	}

	tr.registerFontFaces(func(string) ([]byte, error) {
		return nil, errors.New("boom")
	})

	assert.Empty(t, tr.loadedFonts)
}

func TestRegisterFontFaces_SuccessLoadsFontAndEmitsRow(t *testing.T) {
	t.Parallel()
	tr := &translator{
		sheet: &stylesheet{fontFaces: []fontFaceRule{{family: "myfont", srcURL: "font.ttf"}}},
	}

	tr.registerFontFaces(func(href string) ([]byte, error) {
		assert.Equal(t, "font.ttf", href)
		return []byte{0xDE, 0xAD}, nil
	})

	require.Len(t, tr.loadedFonts, 1)
	assert.Equal(t, "myfont", tr.loadedFonts[0].family)
	assert.Equal(t, []byte{0xDE, 0xAD}, tr.loadedFonts[0].bytes)

	rows := tr.fontRegistrationRows()
	require.Len(t, rows, 1)

	// Rendering the registration row registers the font with the provider.
	p := &lateFontRecordingProvider{}
	rows[0].SetConfig(&entity.Config{MaxGridSize: 12})
	rows[0].Render(p, entity.Cell{Width: 100, Height: 10})
	assert.Equal(t, []string{"myfont"}, p.registered)
}
