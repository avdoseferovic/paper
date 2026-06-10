package pdf

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

// readyPDF returns a PDF instance with a single page added and the embedded
// Helvetica core font selected, ready for drawing and text operations.
func readyPDF(t *testing.T) *PDF {
	t.Helper()
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.AddPage()
	f.SetFont("Helvetica", "", 12)
	if f.Err() {
		t.Fatalf("readyPDF setup failed: %v", f.Error())
	}
	return f
}

// mustOutput renders the document and fails the test if generation errored or
// the result is not a syntactically plausible PDF.
func mustOutput(t *testing.T, f *PDF) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		t.Fatalf("Output failed: %v", err)
	}
	out := buf.Bytes()
	if len(out) < 100 {
		t.Fatalf("output too short: %d bytes", len(out))
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("output missing PDF header: %q", out[:8])
	}
	return out
}

// pngImageBytes returns the encoded bytes of a small RGBA PNG with an alpha
// channel so the soft-mask path is exercised.
func pngImageBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 60), G: uint8(y * 60), B: 128, A: uint8(40 + x*50)})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// jpegImageBytes returns the encoded bytes of a small JPEG image.
func jpegImageBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 60), G: uint8(y * 60), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

// gifImageBytes returns the encoded bytes of a small paletted GIF image.
func gifImageBytes(t *testing.T) []byte {
	t.Helper()
	pal := color.Palette{color.Black, color.White, color.RGBA{R: 255, A: 255}}
	img := image.NewPaletted(image.Rect(0, 0, 4, 4), pal)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetColorIndex(x, y, uint8((x+y)%3))
		}
	}
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode gif: %v", err)
	}
	return buf.Bytes()
}
