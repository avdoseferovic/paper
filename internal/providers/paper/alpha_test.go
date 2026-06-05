package paper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avdoseferovic/paper/v2/internal/providers/paper/gofpdfwrapper"
)

// alphaFpdfStub embeds the Fpdf interface so it satisfies the type with all
// methods inherited (panicking on nil). We override only SetAlpha to record
// calls. Using embedding avoids the import cycle that would come from the
// generated mocks/ package and avoids hand-stubbing the ~180-method interface.
type alphaFpdfStub struct {
	gofpdfwrapper.Fpdf
	calls []alphaCall
}

type alphaCall struct {
	alpha float64
	mode  string
}

func (s *alphaFpdfStub) SetAlpha(alpha float64, mode string) {
	s.calls = append(s.calls, alphaCall{alpha, mode})
}

func TestProvider_WithAlpha_SetsAndRestores(t *testing.T) {
	t.Parallel()
	stub := &alphaFpdfStub{}
	p := &provider{fpdf: stub}
	called := false
	p.WithAlpha(0.5, func() { called = true })
	assert.True(t, called)
	require.Len(t, stub.calls, 2)
	assert.Equal(t, alphaCall{0.5, "Normal"}, stub.calls[0])
	assert.Equal(t, alphaCall{1.0, "Normal"}, stub.calls[1])
}

func TestProvider_WithAlpha_RestoresOnPanic(t *testing.T) {
	t.Parallel()
	stub := &alphaFpdfStub{}
	p := &provider{fpdf: stub}
	require.PanicsWithValue(t, "boom", func() {
		p.WithAlpha(0.3, func() { panic("boom") })
	})
	require.Len(t, stub.calls, 2)
	assert.Equal(t, alphaCall{0.3, "Normal"}, stub.calls[0])
	assert.Equal(t, alphaCall{1.0, "Normal"}, stub.calls[1])
}

func TestProvider_WithAlpha_ClampsOutOfRange(t *testing.T) {
	t.Parallel()
	stub := &alphaFpdfStub{}
	p := &provider{fpdf: stub}
	p.WithAlpha(2.0, func() {})
	require.Len(t, stub.calls, 2)
	assert.Equal(t, 1.0, stub.calls[0].alpha)
	assert.Equal(t, 1.0, stub.calls[1].alpha)
}

func TestProvider_WithAlpha_ClampsNegative(t *testing.T) {
	t.Parallel()
	stub := &alphaFpdfStub{}
	p := &provider{fpdf: stub}
	p.WithAlpha(-1.0, func() {})
	require.Len(t, stub.calls, 2)
	assert.Equal(t, 0.0, stub.calls[0].alpha)
	assert.Equal(t, 1.0, stub.calls[1].alpha)
}
