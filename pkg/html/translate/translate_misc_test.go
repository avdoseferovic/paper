package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/htmllist"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func TestFlexCellContent_GetHeight_SumsChildRows(t *testing.T) {
	t.Parallel()
	content := newFlexCellContent([]core.Row{
		&recordingRow{height: 4},
		&recordingRow{height: 6},
	})
	got := content.GetHeight(nil, &entity.Cell{Width: 100, Height: 100})
	assert.Equal(t, 10.0, got)
}

func TestReverseFlexLogicalRows(t *testing.T) {
	t.Parallel()

	a, b, c := &dom.Node{}, &dom.Node{}, &dom.Node{}
	sa := &css.ComputedStyle{}
	sb := &css.ComputedStyle{}
	sc := &css.ComputedStyle{}
	rows := [][]*dom.Node{{a}, {b}, {c}}
	styles := [][]*css.ComputedStyle{{sa}, {sb}, {sc}}

	reverseFlexLogicalRows(rows, styles)

	assert.Equal(t, [][]*dom.Node{{c}, {b}, {a}}, rows)
	assert.Equal(t, [][]*css.ComputedStyle{{sc}, {sb}, {sa}}, styles)
}

func TestParseCSS_BlocklessAtRuleCollected(t *testing.T) {
	t.Parallel()

	rules := parseCSS(`@import url("extra.css"); p { color: red }`)

	require.NotEmpty(t, rules)
	var found *cssRule
	for _, r := range rules {
		if r.kind == atRule && r.name == "@import" {
			found = r
		}
	}
	require.NotNil(t, found, "expected the @import at-rule to be collected")
	assert.Contains(t, found.prelude, "extra.css")
}

func TestListStyleFromCSS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value  string
		want   htmllist.StyleType
		wantOK bool
	}{
		{value: "", want: "", wantOK: false},
		{value: "none", want: htmllist.None, wantOK: true},
		{value: "disc", want: htmllist.Bullet, wantOK: true},
		{value: "circle", want: htmllist.Bullet, wantOK: true},
		{value: "square", want: htmllist.Bullet, wantOK: true},
		{value: "decimal", want: htmllist.Decimal, wantOK: true},
		{value: "decimal-circle", want: htmllist.DecimalCircle, wantOK: true},
		{value: "lower-alpha", want: htmllist.LowerAlpha, wantOK: true},
		{value: "lower-latin", want: htmllist.LowerAlpha, wantOK: true},
		{value: "upper-alpha", want: htmllist.UpperAlpha, wantOK: true},
		{value: "upper-latin", want: htmllist.UpperAlpha, wantOK: true},
		{value: "lower-roman", want: htmllist.LowerRoman, wantOK: true},
		{value: "upper-roman", want: htmllist.UpperRoman, wantOK: true},
		{value: "fancy-unknown", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run("value "+tt.value, func(t *testing.T) {
			t.Parallel()
			got, ok := listStyleFromCSS(tt.value)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListStyleFromType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  htmllist.StyleType
	}{
		{value: "a", want: htmllist.LowerAlpha},
		{value: "A", want: htmllist.UpperAlpha},
		{value: "i", want: htmllist.LowerRoman},
		{value: "I", want: htmllist.UpperRoman},
		{value: "1", want: htmllist.Decimal},
		{value: "", want: htmllist.Decimal},
	}

	for _, tt := range tests {
		t.Run("type "+tt.value, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, listStyleFromType(tt.value))
		})
	}
}

func TestDecodeCSSDataURI(t *testing.T) {
	t.Parallel()

	t.Run("plain css payload", func(t *testing.T) {
		t.Parallel()
		data, err := decodeCSSDataURI("data:text/css,p{color:red}")
		require.NoError(t, err)
		assert.Equal(t, []byte("p{color:red}"), data)
	})

	t.Run("base64 payload", func(t *testing.T) {
		t.Parallel()
		// "p{}" base64-encoded.
		data, err := decodeCSSDataURI("data:text/css;base64,cHt9")
		require.NoError(t, err)
		assert.Equal(t, []byte("p{}"), data)
	})

	t.Run("missing comma is invalid", func(t *testing.T) {
		t.Parallel()
		_, err := decodeCSSDataURI("data:text/css;base64")
		require.ErrorIs(t, err, errDataURIInvalid)
	})

	t.Run("bad base64 errors", func(t *testing.T) {
		t.Parallel()
		_, err := decodeCSSDataURI("data:text/css;base64,!!!")
		require.Error(t, err)
	})
}

func TestHeadingOutlineLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag    string
		want   int
		wantOK bool
	}{
		{tag: "h1", want: 0, wantOK: true},
		{tag: "h2", want: 1, wantOK: true},
		{tag: "h3", want: 2, wantOK: true},
		{tag: "h4", want: 3, wantOK: true},
		{tag: "h5", want: 4, wantOK: true},
		{tag: "h6", want: 5, wantOK: true},
		{tag: "p", want: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			t.Parallel()
			got, ok := headingOutlineLevel(tt.tag)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
