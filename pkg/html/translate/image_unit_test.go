package translate

import (
	"encoding/base64"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func TestDecodeDataURI(t *testing.T) {
	t.Parallel()

	t.Run("base64 png", func(t *testing.T) {
		t.Parallel()
		payload := []byte{1, 2, 3}
		uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(payload)
		data, ext, err := decodeDataURI(uri)
		require.NoError(t, err)
		assert.Equal(t, payload, data)
		assert.Equal(t, imageExtPNG, ext)
	})

	t.Run("base64 jpeg maps to jpg", func(t *testing.T) {
		t.Parallel()
		uri := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte{9})
		_, ext, err := decodeDataURI(uri)
		require.NoError(t, err)
		assert.Equal(t, imageExtJPG, ext)
	})

	t.Run("plain svg payload", func(t *testing.T) {
		t.Parallel()
		data, ext, err := decodeDataURI("data:image/svg+xml,<svg/>")
		require.NoError(t, err)
		assert.Equal(t, []byte("<svg/>"), data)
		assert.Equal(t, imageExtSVG, ext)
	})

	t.Run("unknown mime defaults to png", func(t *testing.T) {
		t.Parallel()
		_, ext, err := decodeDataURI("data:application/octet-stream,abc")
		require.NoError(t, err)
		assert.Equal(t, imageExtPNG, ext)
	})

	t.Run("missing comma is invalid", func(t *testing.T) {
		t.Parallel()
		_, _, err := decodeDataURI("data:image/png;base64")
		require.ErrorIs(t, err, errDataURIInvalid)
	})

	t.Run("bad base64 payload errors", func(t *testing.T) {
		t.Parallel()
		_, _, err := decodeDataURI("data:image/png;base64,!!!not-base64!!!")
		require.Error(t, err)
	})
}

func TestDecodeDataURIWithLimits_RejectsOversizePayloads(t *testing.T) {
	t.Parallel()

	limits := htmllimits.Limits{MaxImageBytes: 4}

	t.Run("base64 payload over limit", func(t *testing.T) {
		t.Parallel()
		uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("123456789"))
		_, _, err := decodeDataURIWithLimits(uri, limits)
		require.ErrorIs(t, err, htmllimits.ErrImageTooLarge)
	})

	t.Run("plain payload over limit", func(t *testing.T) {
		t.Parallel()
		_, _, err := decodeDataURIWithLimits("data:image/svg+xml,123456789", limits)
		require.ErrorIs(t, err, htmllimits.ErrImageTooLarge)
	})
}

func TestRasterImageSizeMM(t *testing.T) {
	t.Parallel()

	t.Run("malformed data has unknown size and no error", func(t *testing.T) {
		t.Parallel()
		tr := &translator{limits: htmllimits.Default()}
		w, h, err := tr.rasterImageSizeMM([]byte("not an image"))
		require.NoError(t, err)
		assert.Equal(t, 0.0, w)
		assert.Equal(t, 0.0, h)
	})

	t.Run("valid png yields intrinsic size in mm", func(t *testing.T) {
		t.Parallel()
		tr := &translator{limits: htmllimits.Default()}
		w, h, err := tr.rasterImageSizeMM(minimalPNG(t))
		require.NoError(t, err)
		assert.Greater(t, w, 0.0)
		assert.Greater(t, h, 0.0)
	})

	t.Run("pixel count over limit errors", func(t *testing.T) {
		t.Parallel()
		tr := &translator{limits: htmllimits.Limits{MaxImagePixels: 1}}
		_, _, err := tr.rasterImageSizeMM(minimalPNG(t))
		require.ErrorIs(t, err, htmllimits.ErrImageTooLarge)
	})
}

func TestResolveImageDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dims       imageDimensionStyle
		intrinsicW float64
		intrinsicH float64
		fallback   float64
		wantW      float64
		wantH      float64
	}{
		{
			name:  "both explicit kept",
			dims:  imageDimensionStyle{width: 10, height: 20},
			wantW: 10, wantH: 20,
		},
		{
			name:       "width with intrinsic ratio derives height",
			dims:       imageDimensionStyle{width: 10},
			intrinsicW: 100, intrinsicH: 50,
			wantW: 10, wantH: 5,
		},
		{
			name:       "height with intrinsic ratio derives width",
			dims:       imageDimensionStyle{height: 5},
			intrinsicW: 100, intrinsicH: 50,
			wantW: 10, wantH: 5,
		},
		{
			name:  "width only becomes square",
			dims:  imageDimensionStyle{width: 7},
			wantW: 7, wantH: 7,
		},
		{
			name:  "height only becomes square",
			dims:  imageDimensionStyle{height: 9},
			wantW: 9, wantH: 9,
		},
		{
			name:       "intrinsic only used directly",
			intrinsicW: 30, intrinsicH: 40,
			wantW: 30, wantH: 40,
		},
		{
			name:     "nothing falls back to fallback size",
			fallback: 25,
			wantW:    25, wantH: 25,
		},
		{
			name:       "max-width clamps and preserves aspect when height auto",
			dims:       imageDimensionStyle{maxWidth: 10},
			intrinsicW: 100, intrinsicH: 50,
			wantW: 10, wantH: 5,
		},
		{
			name:       "min-width bumps and preserves aspect when height auto",
			dims:       imageDimensionStyle{minWidth: 200},
			intrinsicW: 100, intrinsicH: 50,
			wantW: 200, wantH: 100,
		},
		{
			name:       "max-height clamps and preserves aspect when width auto",
			dims:       imageDimensionStyle{maxHeight: 25},
			intrinsicW: 100, intrinsicH: 50,
			wantW: 50, wantH: 25,
		},
		{
			name:       "min-height bumps and preserves aspect when width auto",
			dims:       imageDimensionStyle{minHeight: 100},
			intrinsicW: 100, intrinsicH: 50,
			wantW: 200, wantH: 100,
		},
		{
			name:  "explicit height not recomputed on max-width clamp",
			dims:  imageDimensionStyle{width: 20, height: 8, maxWidth: 10},
			wantW: 10, wantH: 8,
		},
		{
			name:  "explicit width not recomputed on max-height clamp",
			dims:  imageDimensionStyle{width: 20, height: 8, maxHeight: 4},
			wantW: 20, wantH: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, h := resolveImageDimensions(tt.dims, tt.intrinsicW, tt.intrinsicH, tt.fallback)
			assert.InDelta(t, tt.wantW, w, 0.001)
			assert.InDelta(t, tt.wantH, h, 0.001)
		})
	}
}

func TestTranslatorUnsupported(t *testing.T) {
	t.Parallel()

	t.Run("forwards to handler", func(t *testing.T) {
		t.Parallel()
		var gotKind, gotMsg string
		tr := &translator{unsupportedHandler: func(kind, msg string) {
			gotKind, gotMsg = kind, msg
		}}
		tr.unsupported("img.src", "boom")
		assert.Equal(t, "img.src", gotKind)
		assert.Equal(t, "boom", gotMsg)
	})

	t.Run("nil handler is safe", func(t *testing.T) {
		t.Parallel()
		tr := &translator{}
		tr.unsupported("img.src", "boom") // must not panic
	})
}

func TestExtensionType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want extension.Type
	}{
		{in: "png", want: extension.Png},
		{in: "jpg", want: extension.Jpg},
		{in: "jpeg", want: extension.Jpg},
		{in: "JPEG", want: extension.Jpg},
		{in: "PNG", want: extension.Png},
		{in: "gif", want: ""},
		{in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run("ext "+tt.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, extensionType(tt.in))
		})
	}
}

func TestImageCols(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		widthMM   float64
		cellWidth float64
		gridSize  int
		want      int
	}{
		{name: "zero grid uses default and full width", widthMM: 0, cellWidth: 100, gridSize: 0, want: defaultGridSize},
		{name: "zero width takes full grid", widthMM: 0, cellWidth: 100, gridSize: 12, want: 12},
		{name: "zero cell width takes full grid", widthMM: 50, cellWidth: 0, gridSize: 12, want: 12},
		{name: "half width rounds to half the grid", widthMM: 50, cellWidth: 100, gridSize: 12, want: 6},
		{name: "tiny image clamps to one col", widthMM: 1, cellWidth: 1000, gridSize: 12, want: 1},
		{name: "oversize image clamps to grid", widthMM: 500, cellWidth: 100, gridSize: 12, want: 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, imageCols(tt.widthMM, tt.cellWidth, tt.gridSize))
		})
	}
}

func TestExtFromFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "photo.png", want: "png"},
		{in: "photo.JPEG", want: imageExtJPG},
		{in: "photo.jpeg", want: imageExtJPG},
		{in: "photo.jpg", want: "jpg"},
		{in: "drawing.svg", want: "svg"},
		{in: "noext", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, extFromFilename(tt.in))
		})
	}
}

func TestParseImageDimension(t *testing.T) {
	t.Parallel()

	t.Run("empty is zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, parseImageDimension(""))
	})

	t.Run("unitless treated as px", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, css.ParseLength("20px", 0), parseImageDimension("20"))
	})

	t.Run("mm passes through", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 10.0, parseImageDimension("10mm"))
	})
}

func TestAltRowStyled(t *testing.T) {
	t.Parallel()

	imgNode := func(t *testing.T, src string) *dom.Node {
		t.Helper()
		doc, err := dom.Parse(src)
		require.NoError(t, err)
		n := findNode(doc, "img")
		require.NotNil(t, n)
		return n
	}

	t.Run("empty alt yields no rows", func(t *testing.T) {
		t.Parallel()
		n := imgNode(t, `<html><body><img src="x.png" alt="  "></body></html>`)
		assert.Nil(t, altRowStyled(n, nil))
	})

	t.Run("alt text yields one paragraph row", func(t *testing.T) {
		t.Parallel()
		n := imgNode(t, `<html><body><img src="x.png" alt="fallback"></body></html>`)
		rows := altRowStyled(n, nil)
		require.Len(t, rows, 1)
	})
}

func TestBaseDirResolver_DataURIBypassesFileSystem(t *testing.T) {
	t.Parallel()

	resolver := baseDirResolver(t.TempDir())

	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte{1})
	data, ext, err := resolver(uri)
	require.NoError(t, err)
	assert.Equal(t, []byte{1}, data)
	assert.Equal(t, imageExtPNG, ext)
}
