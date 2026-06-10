package html_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html"
)

func TestFromString(t *testing.T) {
	t.Parallel()

	t.Run("returns rows for simple html", func(t *testing.T) {
		t.Parallel()
		rows, err := html.FromString("<p>hello</p>")
		require.NoError(t, err)
		assert.NotEmpty(t, rows)
	})

	t.Run("handles malformed html without error", func(t *testing.T) {
		t.Parallel()
		rows, err := html.FromString("<p>unclosed")
		require.NoError(t, err) // golang.org/x/net/html is permissive
		assert.NotEmpty(t, rows)
	})

	t.Run("empty string returns empty rows", func(t *testing.T) {
		t.Parallel()
		rows, err := html.FromString("")
		require.NoError(t, err)
		_ = rows // may be nil or empty
	})
}

func TestFromReader(t *testing.T) {
	t.Parallel()
	r := strings.NewReader("<p>hi</p>")
	rows, err := html.FromReader(r)
	require.NoError(t, err)
	assert.NotEmpty(t, rows)
}

func TestFromStringCtx_ReturnsContextError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rows, err := html.FromStringCtx(ctx, "<p>hi</p>")

	assert.Nil(t, rows)
	require.ErrorIs(t, err, context.Canceled)
}

func TestFromReaderCtx_ReturnsContextError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rows, err := html.FromReaderCtx(ctx, strings.NewReader("<p>hi</p>"))

	assert.Nil(t, rows)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithUnsupportedHandler is invocable", func(t *testing.T) {
		t.Parallel()
		opt := html.WithUnsupportedHandler(func(_, _ string) {})
		assert.NotNil(t, opt)
	})
}

func TestFromString_RejectsOversizeImagePixels(t *testing.T) {
	t.Parallel()

	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngHeaderWithDimensions(40_000, 40_000))
	_, err := html.FromString(`<html><body><img src="` + uri + `"></body></html>`)

	require.ErrorIs(t, err, html.ErrImageTooLarge)
}

func TestFromString_RejectsOversizeImageBytes(t *testing.T) {
	t.Parallel()

	uri := "data:image/png;base64," + strings.Repeat("A", 24)
	_, err := html.FromString(
		`<html><body><img src="`+uri+`"></body></html>`,
		html.WithLimits(html.Limits{MaxImageBytes: 10}),
	)

	require.ErrorIs(t, err, html.ErrImageTooLarge)
}

func TestFromString_RejectsDeepDOM(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	b.WriteString("<html><body>")
	for range 10_000 {
		b.WriteString("<div>")
	}
	b.WriteString("x")
	for range 10_000 {
		b.WriteString("</div>")
	}
	b.WriteString("</body></html>")

	_, err := html.FromString(b.String())

	require.ErrorIs(t, err, html.ErrDOMTooDeep)
}

func TestFromString_RejectsLargeDOM(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	b.WriteString("<html><body>")
	for range 20 {
		b.WriteString("<span>x</span>")
	}
	b.WriteString("</body></html>")

	_, err := html.FromString(b.String(), html.WithLimits(html.Limits{MaxDOMNodes: 10}))

	require.ErrorIs(t, err, html.ErrDOMTooLarge)
}

func TestFromString_RejectsTooManyStyleRules(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	b.WriteString("<html><head><style>")
	for i := range 20 {
		b.WriteString(".x")
		b.WriteString(string(rune('a' + i)))
		b.WriteString("{color:red}")
	}
	b.WriteString("</style></head><body><p>ok</p></body></html>")

	_, err := html.FromString(b.String(), html.WithLimits(html.Limits{MaxStyleRules: 5}))

	require.ErrorIs(t, err, html.ErrStyleRulesTooLarge)
}

func TestFromString_RejectsOversizeSVG(t *testing.T) {
	t.Parallel()

	_, err := html.FromString(
		`<html><body><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000"><rect width="10" height="10"/></svg></body></html>`,
		html.WithLimits(html.Limits{MaxSVGPixels: 100}),
	)

	require.ErrorIs(t, err, html.ErrSVGTooLarge)
}

func pngHeaderWithDimensions(width, height uint32) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	writePNGChunk(&buf, "IHDR", func() []byte {
		data := make([]byte, 13)
		binary.BigEndian.PutUint32(data[0:4], width)
		binary.BigEndian.PutUint32(data[4:8], height)
		data[8] = 8 // bit depth
		data[9] = 6 // truecolour with alpha
		return data
	}())
	writePNGChunk(&buf, "IEND", nil)
	return buf.Bytes()
}

func writePNGChunk(buf *bytes.Buffer, name string, data []byte) {
	_ = binary.Write(buf, binary.BigEndian, uint32(len(data)))
	buf.WriteString(name)
	buf.Write(data)
	crc := crc32.NewIEEE()
	_, _ = crc.Write([]byte(name))
	_, _ = crc.Write(data)
	_ = binary.Write(buf, binary.BigEndian, crc.Sum32())
}
