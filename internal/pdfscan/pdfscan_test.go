package pdfscan

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestSkipValue(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		in   string
		want string // the bytes left after the value
	}{
		"dictionary":            {"<< /A 1 >> tail", " tail"},
		"nested dictionary":     {"<< /A << /B 2 >> >> tail", " tail"},
		"dictionary with paren": {"<< /A (>>) >> tail", " tail"},
		"indirect reference":    {"12 0 R tail", " tail"},
		"array":                 {"[1 2 [3]] tail", " tail"},
		"array with paren":      {"[(])] tail", " tail"},
		"literal string":        {"(a(b)c) tail", " tail"},
		"escaped paren":         {`(a\)b) tail`, " tail"},
		"bare token":            {"Name tail", " tail"},
		// '/' is a delimiter, so a name token is consumed as "/" then "Name".
		"name delimiter":    {"/Name tail", "Name tail"},
		"number":            {"3.14 tail", " tail"},
		"leading space":     {"   42 tail", " tail"},
		"unterminated dict": {"<< /A 1", ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data := []byte(tc.in)
			assert.Equal(t, tc.want, string(data[SkipValue(data, 0):]))
		})
	}
}

func TestSkipIndirectRef_RejectsNonReferences(t *testing.T) {
	t.Parallel()

	for name, in := range map[string]string{
		"single number": "12 R",
		"not a number":  "/A 0 R",
		"missing R":     "12 0 /A",
		"empty":         "",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			end, ok := SkipIndirectRef([]byte(in), 0)
			assert.False(t, ok)
			assert.Equal(t, 0, end)
		})
	}
}

func TestParseLiteral_UnescapesAndKeepsNesting(t *testing.T) {
	t.Parallel()

	value, end := ParseLiteral([]byte(`(Jane \(J\) Doe)rest`), 0)
	assert.Equal(t, "Jane (J) Doe", value)
	assert.Equal(t, len(`(Jane \(J\) Doe)`), end)

	value, _ = ParseLiteral([]byte("(outer (inner) done)"), 0)
	assert.Equal(t, "outer (inner) done", value)
}

func TestSkipLiteral_MatchesParseLiteralOffset(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"(plain)", `(esc \) here)`, "(nest (ed))", "(unterminated"} {
		_, want := ParseLiteral([]byte(in), 0)
		assert.Equal(t, want, SkipLiteral([]byte(in), 0), in)
	}
}

func TestSkipToken_AlwaysAdvances(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 4, SkipToken([]byte("Name (x)"), 0))
	// A name's leading '/' is a delimiter and is consumed on its own.
	assert.Equal(t, 1, SkipToken([]byte("/Name"), 0))
	// A lone delimiter still consumes one byte so callers cannot spin.
	assert.Equal(t, 1, SkipToken([]byte("[[["), 0))
	assert.Equal(t, 0, SkipToken([]byte(""), 0))
}

func TestSkipSpaces_StopsAtContent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 4, SkipSpaces([]byte(" \t\n\r/A"), 0))
	assert.Equal(t, 3, SkipSpaces([]byte("   "), 0))
}

func TestKeyIndex(t *testing.T) {
	t.Parallel()

	dict := []byte("<< /Type /Page /Rotate 90 /MediaBox [0 0 1 1] >>")
	assert.Equal(t, 3, KeyIndex(dict, "Type"))
	assert.Equal(t, 15, KeyIndex(dict, "Rotate"))
	assert.Equal(t, -1, KeyIndex(dict, "Missing"))
	// A prefix of a longer name must not match.
	assert.Equal(t, -1, KeyIndex(dict, "Rot"))
	assert.Equal(t, -1, KeyIndex(dict, "Med"))
}

func TestRemoveDictEntry(t *testing.T) {
	t.Parallel()

	dict := []byte("<< /Type /Page /Rotate 90 /MediaBox [0 0 1 1] >>")
	assert.Equal(t, "<< /Type /Page /MediaBox [0 0 1 1] >>", string(RemoveDictEntry(dict, "Rotate")))
	assert.Equal(t, "<< /Type /Page /Rotate 90 >>", string(RemoveDictEntry(dict, "MediaBox")))
	assert.Equal(t, string(dict), string(RemoveDictEntry(dict, "Absent")))
}
