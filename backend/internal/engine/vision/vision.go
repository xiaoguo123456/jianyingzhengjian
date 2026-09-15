// Package vision wraps the narrow per-call vision APIs with retries (docs/GENERATION_PIPELINE.md §2).
package vision

import (
	"context"
	"image"
	"time"

	"yingji/backend/internal/provider/face"
	"yingji/backend/internal/provider/matting"
)

type Engine struct {
	det face.Detector
	cmp face.Comparer
	mat matting.Matter
}

func New(det face.Detector, cmp face.Comparer, mat matting.Matter) *Engine {
	return &Engine{det: det, cmp: cmp, mat: mat}
}

func retry[T any](ctx context.Context, n int, fn func() (T, error)) (T, error) {
	var last error
	var zero T
	for i := 0; i <= n; i++ {
		v, err := fn()
		if err == nil {
			return v, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(time.Duration(300*(i+1)) * time.Millisecond):
		}
	}
	return zero, last
}

func (e *Engine) Detect(ctx context.Context, img image.Image, raw []byte) (face.Result, error) {
	return retry(ctx, 2, func() (face.Result, error) { return e.det.Detect(ctx, img, raw) })
}

func (e *Engine) Compare(ctx context.Context, a, b []byte) (float64, error) {
	if e.cmp == nil {
		return 1, nil
	}
	return retry(ctx, 2, func() (float64, error) { return e.cmp.Compare(ctx, a, b) })
}

func (e *Engine) Matte(ctx context.Context, img image.Image, raw []byte, f face.Result) (*image.Alpha, error) {
	return retry(ctx, 2, func() (*image.Alpha, error) { return e.mat.Matte(ctx, img, raw, f) })
}
