package genmodel

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math/rand"
	"time"

	"github.com/disintegration/imaging"
)

// Mock returns the source image with a tinted band so a generated result is
// distinguishable from the input. Simulates latency; costs nothing.
type Mock struct {
	Latency time.Duration
}

func (Mock) Name() string { return "mock" }

func (Mock) Capabilities() Caps {
	return Caps{Img2Img: true, Reference: true, Edit: true, MaskEdit: true, MaxSide: 4096}
}

func (m Mock) Run(ctx context.Context, req Request) (Result, error) {
	if m.Latency > 0 {
		select {
		case <-time.After(m.Latency):
		case <-ctx.Done():
			return Result{}, ctx.Err()
		}
	}
	src, err := imaging.Decode(bytes.NewReader(req.Source), imaging.AutoOrientation(true))
	if err != nil {
		return Result{}, fmt.Errorf("mock gen: decode: %w", err)
	}
	seed := req.Seed
	if seed == 0 {
		seed = rand.Int63()
	}
	out := imaging.Clone(src)
	if req.Width > 0 && req.Height > 0 {
		out = imaging.Fill(out, req.Width, req.Height, imaging.Center, imaging.Lanczos)
	}
	// tint band at the bottom whose hue depends on the seed
	b := out.Bounds()
	band := image.Rect(b.Min.X, b.Max.Y-b.Dy()/14, b.Max.X, b.Max.Y)
	c := color.NRGBA{R: uint8(60 + seed%120), G: uint8(120 + (seed/7)%100), B: 230, A: 150}
	draw.Draw(out, band, &image.Uniform{c}, image.Point{}, draw.Over)
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, out, imaging.JPEG, imaging.JPEGQuality(92)); err != nil {
		return Result{}, err
	}
	return Result{Image: buf.Bytes(), Format: "jpeg", CostCents: 0, ProviderRef: fmt.Sprintf("mock-%d", seed), Seed: seed}, nil
}
