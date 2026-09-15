package face

import (
	"context"
	"image"
)

// Mock assumes one centred face; used in dev, CI and the offline pipeline.
type Mock struct{}

func (Mock) Detect(ctx context.Context, img image.Image, raw []byte) (Result, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	fw := int(float64(w) * 0.34)
	fh := int(float64(h) * 0.30)
	cx := b.Min.X + w/2
	top := b.Min.Y + int(float64(h)*0.22)
	return Result{Faces: 1, Box: image.Rect(cx-fw/2, top, cx+fw/2, top+fh), Quality: 0.9, Gender: ""}, nil
}

func (Mock) Compare(ctx context.Context, a, b []byte) (float64, error) { return 0.92, nil }
