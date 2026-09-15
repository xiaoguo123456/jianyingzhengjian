package matting

import (
	"context"
	"image"
	"image/color"
	"testing"

	"yingji/backend/internal/provider/face"
)

// scene draws a subject (dark blob with a light "collar" touching the bottom edge)
// on an even light background, mimicking an ID-photo source.
func scene(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	bg := color.NRGBA{R: 220, G: 228, B: 238, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, bg)
		}
	}
	// head: dark rectangle in the middle
	for y := h / 5; y < h*3/5; y++ {
		for x := w / 3; x < w*2/3; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 60, G: 40, B: 35, A: 255})
		}
	}
	// collar: near-white, touching the bottom edge, close to the background colour
	for y := h * 3 / 5; y < h; y++ {
		for x := w / 4; x < w*3/4; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 246, G: 248, B: 250, A: 255})
		}
	}
	return img
}

func TestMockMatteKeepsSubjectAndRemovesBackground(t *testing.T) {
	const w, h = 200, 300
	img := scene(w, h)
	m, err := Mock{}.Matte(context.Background(), img, nil, face.Result{Faces: 1, Box: image.Rect(w/3, h/5, w*2/3, h*3/5)})
	if err != nil {
		t.Fatalf("matte: %v", err)
	}
	// a top corner is background
	if a := m.AlphaAt(2, 2).A; a > 40 {
		t.Errorf("top-left alpha = %d, want transparent background", a)
	}
	// the head is subject
	if a := m.AlphaAt(w/2, h/3).A; a < 200 {
		t.Errorf("head alpha = %d, want opaque subject", a)
	}
	// the collar touches the bottom edge and is near-background in colour: it must be kept
	if a := m.AlphaAt(w/2, h-5).A; a < 200 {
		t.Errorf("collar alpha = %d, want opaque (bottom edge must not seed the fill)", a)
	}
}

// A busy image where keying cannot separate anything must keep the whole frame
// rather than returning a mask that would mangle the photo.
func TestMockMatteFallsBackToFullFrame(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 60, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 60; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 4), G: uint8(y * 4), B: 90, A: 255})
		}
	}
	m, err := Mock{}.Matte(context.Background(), img, nil, face.Result{})
	if err != nil {
		t.Fatalf("matte: %v", err)
	}
	for _, p := range []image.Point{{2, 2}, {30, 30}, {58, 58}} {
		if a := m.AlphaAt(p.X, p.Y).A; a != 255 {
			t.Errorf("alpha at %v = %d, want 255 (full-frame fallback)", p, a)
		}
	}
}
