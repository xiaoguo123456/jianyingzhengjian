// Package face defines the face detection / comparison providers (vision engine).
package face

import (
	"context"
	"image"
)

type Result struct {
	Faces    int
	Box      image.Rectangle // bbox of the primary face in image pixels (chin-to-brow region)
	Quality  float64         // 0..1
	Gender   string          // "male" | "female" | ""
	Occluded bool
}

type Detector interface {
	Detect(ctx context.Context, img image.Image, raw []byte) (Result, error)
}

type Comparer interface {
	// Compare returns a 0..1 similarity between the primary faces in a and b.
	Compare(ctx context.Context, a, b []byte) (float64, error)
}
