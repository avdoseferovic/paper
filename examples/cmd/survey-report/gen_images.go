//go:build ignore
// +build ignore

// gen_images.go generates synthetic "medical" placeholder images for the
// survey-report demo so the demo is self-contained.
//
// Run: cd examples && go run ./cmd/survey-report/gen_images.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/avdoseferovic/paper/examples/internal/examplepath"
)

func main() {
	assetsDir := examplepath.Module("cmd/survey-report/assets")
	if err := writePNG(filepath.Join(assetsDir, "chest-xray.png"), chestXRay(440, 520)); err != nil {
		log.Fatalf("chest-xray: %v", err)
	}
	if err := writePNG(filepath.Join(assetsDir, "ecg-strip.png"), ecgStrip(800, 220)); err != nil {
		log.Fatalf("ecg-strip: %v", err)
	}
	if err := writePNG(filepath.Join(assetsDir, "vitals-chart.png"), vitalsChart(640, 320)); err != nil {
		log.Fatalf("vitals: %v", err)
	}
	log.Println("generated 3 images in", assetsDir)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// chestXRay renders a stylised greyscale image evocative of a PA chest X-ray:
// dark background, brighter central column for the spine, lighter "lung field"
// ovals on either side, with a softer "cardiac silhouette" overlay on the left.
func chestXRay(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)/2, float64(h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			// Base radial gradient from light center to dark edge.
			r := math.Hypot(dx, dy) / math.Hypot(cx, cy)
			base := 0.85 - 0.7*r // 0.85 center → 0.15 corners

			// Spine column — bright vertical band, ~12% width.
			spine := math.Exp(-math.Pow(dx/(float64(w)*0.06), 2))
			base += 0.20 * spine

			// Left and right lung fields — darker ovals roughly 25% off-center.
			lungR := math.Hypot((dx-float64(w)*0.18)/(float64(w)*0.22), (dy-float64(h)*0.05)/(float64(h)*0.30))
			lungL := math.Hypot((dx+float64(w)*0.18)/(float64(w)*0.22), (dy-float64(h)*0.05)/(float64(h)*0.30))
			lung := math.Min(lungR, lungL)
			base -= 0.18 * math.Exp(-math.Pow(lung, 2))

			// Cardiac silhouette — softer light blob slightly left of centre.
			heart := math.Hypot((dx+float64(w)*0.05)/(float64(w)*0.12), (dy-float64(h)*0.10)/(float64(h)*0.18))
			base += 0.10 * math.Exp(-math.Pow(heart, 2))

			// Ribcage suggestion — gentle horizontal striations.
			rib := math.Sin(float64(y)*0.18) * 0.06 * math.Exp(-math.Pow((float64(x)-cx)/(float64(w)*0.45), 2))
			base += rib

			// Subtle film grain via deterministic noise.
			noise := (math.Sin(float64(x)*12.9898+float64(y)*78.233)*43758.5453 -
				math.Floor(math.Sin(float64(x)*12.9898+float64(y)*78.233)*43758.5453)) - 0.5
			base += noise * 0.04

			v := clamp01(base)
			gray := uint8(math.Round(v * 255))
			img.Set(x, y, color.RGBA{gray, gray, gray, 255})
		}
	}
	return img
}

// ecgStrip renders a stylised ECG strip: pink grid + a normal sinus rhythm
// waveform traced in dark red.
func ecgStrip(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	bg := color.RGBA{255, 244, 244, 255}
	gridFine := color.RGBA{246, 188, 188, 255}
	gridCoarse := color.RGBA{220, 140, 140, 255}
	waveCol := color.RGBA{135, 25, 25, 255}

	// Fill background.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, bg)
		}
	}

	// 1mm fine grid every 8px, 5mm coarse grid every 40px.
	for x := 0; x < w; x++ {
		if x%40 == 0 {
			drawVLine(img, x, 0, h, gridCoarse)
		} else if x%8 == 0 {
			drawVLine(img, x, 0, h, gridFine)
		}
	}
	for y := 0; y < h; y++ {
		if y%40 == 0 {
			drawHLine(img, 0, w, y, gridCoarse)
		} else if y%8 == 0 {
			drawHLine(img, 0, w, y, gridFine)
		}
	}

	// Trace one sinus complex repeated across the strip.
	baseline := float64(h) * 0.55
	period := 200.0 // pixels per beat (~50 mm at 8px/mm = 60 bpm @ 25mm/s strip)
	for x := 0; x < w-1; x++ {
		y1 := ecgY(float64(x), period, baseline, float64(h))
		y2 := ecgY(float64(x+1), period, baseline, float64(h))
		drawThickLine(img, x, int(math.Round(y1)), x+1, int(math.Round(y2)), waveCol, 2)
	}

	return img
}

// ecgY returns the Y position of an ECG trace at a given x.
func ecgY(x, period, baseline, totalH float64) float64 {
	phase := math.Mod(x, period) / period // 0..1 within one cardiac cycle
	switch {
	case phase < 0.08: // P wave
		return baseline - math.Sin(phase/0.08*math.Pi)*totalH*0.05
	case phase < 0.18: // PR segment (flat)
		return baseline
	case phase < 0.22: // Q dip
		return baseline + math.Sin((phase-0.18)/0.04*math.Pi)*totalH*0.05
	case phase < 0.26: // R spike up
		return baseline - math.Sin((phase-0.22)/0.04*math.Pi)*totalH*0.40
	case phase < 0.30: // S dip
		return baseline + math.Sin((phase-0.26)/0.04*math.Pi)*totalH*0.10
	case phase < 0.38: // ST segment
		return baseline
	case phase < 0.55: // T wave
		return baseline - math.Sin((phase-0.38)/0.17*math.Pi)*totalH*0.10
	default:
		return baseline
	}
}

// vitalsChart renders a stylised line chart of "Blood pressure / heart rate
// over the last 7 days" — clean white background, gentle grid, two trend lines.
func vitalsChart(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := color.RGBA{252, 252, 254, 255}
	grid := color.RGBA{215, 222, 232, 255}
	axis := color.RGBA{120, 130, 145, 255}
	sys := color.RGBA{192, 47, 47, 255}
	hr := color.RGBA{52, 110, 196, 255}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, bg)
		}
	}

	// Margins.
	left, right, top, bottom := 50, 20, 20, 30
	innerW := w - left - right
	innerH := h - top - bottom

	// Grid (6 vertical, 5 horizontal).
	for i := 1; i < 7; i++ {
		x := left + i*innerW/7
		drawVLine(img, x, top, h-bottom, grid)
	}
	for i := 1; i < 5; i++ {
		y := top + i*innerH/5
		drawHLine(img, left, w-right, y, grid)
	}

	// Axes.
	drawVLine(img, left, top, h-bottom, axis)
	drawHLine(img, left, w-right, h-bottom, axis)

	// Synthesise 7-day series (Mon..Sun).
	sysPts := []float64{132, 128, 134, 129, 138, 130, 124}
	hrPts := []float64{82, 78, 84, 80, 86, 79, 76}

	plot := func(points []float64, low, high float64, c color.RGBA) {
		for i := 0; i+1 < len(points); i++ {
			x1 := left + i*innerW/(len(points)-1)
			x2 := left + (i+1)*innerW/(len(points)-1)
			y1 := top + int(math.Round((1-(points[i]-low)/(high-low))*float64(innerH)))
			y2 := top + int(math.Round((1-(points[i+1]-low)/(high-low))*float64(innerH)))
			drawThickLine(img, x1, y1, x2, y2, c, 2)
		}
		// Dots at each data point.
		for i, p := range points {
			x := left + i*innerW/(len(points)-1)
			y := top + int(math.Round((1-(p-low)/(high-low))*float64(innerH)))
			drawDot(img, x, y, 3, c)
		}
	}

	plot(sysPts, 110, 145, sys)
	plot(hrPts, 60, 95, hr)

	return img
}

// ── drawing helpers ──────────────────────────────────────────────────────────

func drawVLine(img *image.RGBA, x, y0, y1 int, c color.Color) {
	for y := y0; y < y1; y++ {
		if inBounds(img, x, y) {
			img.Set(x, y, c)
		}
	}
}

func drawHLine(img *image.RGBA, x0, x1, y int, c color.Color) {
	for x := x0; x < x1; x++ {
		if inBounds(img, x, y) {
			img.Set(x, y, c)
		}
	}
}

func drawThickLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color, thickness int) {
	// Bresenham + perpendicular thickness.
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := step(x0, x1)
	sy := step(y0, y1)
	err := dx + dy
	for {
		for ty := -thickness / 2; ty <= thickness/2; ty++ {
			for tx := -thickness / 2; tx <= thickness/2; tx++ {
				if inBounds(img, x0+tx, y0+ty) {
					img.Set(x0+tx, y0+ty, c)
				}
			}
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawDot(img *image.RGBA, cx, cy, radius int, c color.Color) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius && inBounds(img, cx+dx, cy+dy) {
				img.Set(cx+dx, cy+dy, c)
			}
		}
	}
}

func inBounds(img *image.RGBA, x, y int) bool {
	b := img.Bounds()
	return x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func step(a, b int) int {
	if a < b {
		return 1
	}
	if a > b {
		return -1
	}
	return 0
}
