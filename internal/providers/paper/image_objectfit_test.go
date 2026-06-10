package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestObjectImageRect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fit      string
		position string
		imageW   float64
		imageH   float64
		boxW     float64
		boxH     float64
		wantX    float64
		wantY    float64
		wantW    float64
		wantH    float64
	}{
		{
			name:   "contain centers preserved aspect image inside box",
			fit:    "contain",
			imageW: 100, imageH: 50, boxW: 40, boxH: 40,
			wantX: 0, wantY: 10, wantW: 40, wantH: 20,
		},
		{
			name:     "cover clips overflow and honors right bottom position",
			fit:      "cover",
			position: "right bottom",
			imageW:   100, imageH: 50, boxW: 40, boxH: 40,
			wantX: -40, wantY: 0, wantW: 80, wantH: 40,
		},
		{
			name:   "fill stretches to object box",
			fit:    "fill",
			imageW: 100, imageH: 50, boxW: 40, boxH: 40,
			wantX: 0, wantY: 0, wantW: 40, wantH: 40,
		},
		{
			name:   "none uses intrinsic size and centers by default",
			fit:    "none",
			imageW: 100, imageH: 50, boxW: 40, boxH: 40,
			wantX: -30, wantY: -5, wantW: 100, wantH: 50,
		},
		{
			name:   "scale-down keeps small intrinsic image",
			fit:    "scale-down",
			imageW: 30, imageH: 20, boxW: 40, boxH: 40,
			wantX: 5, wantY: 10, wantW: 30, wantH: 20,
		},
		{
			name:   "scale-down contains large intrinsic image",
			fit:    "scale-down",
			imageW: 100, imageH: 50, boxW: 40, boxH: 40,
			wantX: 0, wantY: 10, wantW: 40, wantH: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := objectImageRect(tt.fit, tt.position, tt.imageW, tt.imageH, 0, 0, tt.boxW, tt.boxH)
			assert.InDelta(t, tt.wantX, got.X, 0.001)
			assert.InDelta(t, tt.wantY, got.Y, 0.001)
			assert.InDelta(t, tt.wantW, got.Width, 0.001)
			assert.InDelta(t, tt.wantH, got.Height, 0.001)
		})
	}
}
