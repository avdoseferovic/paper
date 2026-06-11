package pdf

import (
	"bytes"
	"testing"
)

func TestImageTypeFromMime(t *testing.T) {
	f := NewCustom(&InitType{})
	cases := map[string]string{
		"image/jpeg": "jpg",
		"image/png":  "png",
		"image/gif":  "gif",
	}
	for mime, want := range cases {
		if got := f.ImageTypeFromMime(mime); got != want {
			t.Errorf("ImageTypeFromMime(%q) = %q, want %q", mime, got, want)
		}
	}
}

func TestImageTypeFromMimeUnknownErrors(t *testing.T) {
	f := NewCustom(&InitType{})
	f.ImageTypeFromMime("application/octet-stream")
	if !f.Err() {
		t.Fatal("expected error for unknown mime type")
	}
}

func TestRegisterAndRenderPNG(t *testing.T) {
	f := readyPDF(t)
	info := f.RegisterImageOptionsReader("logo", ImageOptions{ImageType: "png"}, bytes.NewReader(pngImageBytes(t)))
	if f.Err() {
		t.Fatalf("register png errored: %v", f.Error())
	}
	if info == nil || info.Width() <= 0 {
		t.Fatalf("unexpected image info: %+v", info)
	}
	// GetImageInfo should return the same registered image.
	if got := f.GetImageInfo("logo"); got == nil {
		t.Fatal("GetImageInfo returned nil for registered image")
	}
	f.ImageOptions("logo", 10, 10, 40, 0, false, ImageOptions{ImageType: "png"}, 0, "")
	if f.Err() {
		t.Fatalf("render png errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestRegisterAndRenderJPEG(t *testing.T) {
	f := readyPDF(t)
	f.RegisterImageReader("photo", "jpg", bytes.NewReader(jpegImageBytes(t)))
	if f.Err() {
		t.Fatalf("register jpeg errored: %v", f.Error())
	}
	f.Image("photo", 10, 60, 40, 0, false, "jpg", 0, "")
	if f.Err() {
		t.Fatalf("render jpeg errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestRegisterAndRenderGIF(t *testing.T) {
	f := readyPDF(t)
	f.RegisterImageReader("anim", "gif", bytes.NewReader(gifImageBytes(t)))
	if f.Err() {
		t.Fatalf("register gif errored: %v", f.Error())
	}
	f.Image("anim", 10, 110, 40, 0, false, "gif", 0, "")
	if f.Err() {
		t.Fatalf("render gif errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestImageWithFlowAdvancesCursor(t *testing.T) {
	f := readyPDF(t)
	f.RegisterImageReader("flow", "png", bytes.NewReader(pngImageBytes(t)))
	yBefore := f.GetY()
	f.Image("flow", 10, 10, 30, 30, true, "png", 0, "")
	if f.GetY() <= yBefore {
		t.Fatal("flow=true should advance the Y cursor below the image")
	}
}

func TestGetImageInfoUnknownReturnsNil(t *testing.T) {
	f := NewCustom(&InitType{})
	if got := f.GetImageInfo("does-not-exist"); got != nil {
		t.Fatalf("expected nil for unknown image, got %+v", got)
	}
}

func TestRegisterImageOptionsReaderRequiresTypeForCustomReader(t *testing.T) {
	f := readyPDF(t)
	// With no explicit type, a custom reader cannot be auto-detected.
	f.RegisterImageOptionsReader("auto", ImageOptions{}, bytes.NewReader(pngImageBytes(t)))
	if !f.Err() {
		t.Fatal("expected error when type omitted for a custom reader")
	}
}
