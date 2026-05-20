package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputedStyle_LetterSpacing(t *testing.T) {
	t.Parallel()
	s := NewComputedStyle()
	s.Apply("letter-spacing", "0.5pt", nil)
	assert.InDelta(t, 0.176389, s.LetterSpacing, 0.001) // 0.5pt → mm
}

func TestComputedStyle_TextTransform(t *testing.T) {
	t.Parallel()
	s := NewComputedStyle()
	s.Apply("text-transform", "uppercase", nil)
	assert.Equal(t, "uppercase", s.TextTransform)

	s2 := NewComputedStyle()
	s2.Apply("text-transform", "capitalize", nil)
	assert.Equal(t, "capitalize", s2.TextTransform)
}

func TestComputedStyle_TextIndent(t *testing.T) {
	t.Parallel()
	s := NewComputedStyle()
	s.Apply("text-indent", "5mm", nil)
	assert.InDelta(t, 5.0, s.TextIndent, 0.001)
}

func TestComputedStyle_WhiteSpace(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"normal", "nowrap", "pre", "pre-wrap", "pre-line"} {
		s := NewComputedStyle()
		s.Apply("white-space", v, nil)
		assert.Equal(t, v, s.WhiteSpace, "white-space: %s", v)
	}
}

func TestApplyTextTransform(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "HELLO", ApplyTextTransform("hello", "uppercase"))
	assert.Equal(t, "hello", ApplyTextTransform("HELLO", "lowercase"))
	assert.Equal(t, "Hello World", ApplyTextTransform("hello world", "capitalize"))
	assert.Equal(t, "Mixed Case Text", ApplyTextTransform("mixed case text", "capitalize"))
	assert.Equal(t, "hello", ApplyTextTransform("hello", "none"))
	assert.Equal(t, "hello", ApplyTextTransform("hello", ""))
}
