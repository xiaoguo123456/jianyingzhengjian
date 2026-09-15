// Package matting defines portrait segmentation providers.
package matting

import (
	"context"
	"image"

	"yingji/backend/internal/provider/face"
)

type Matter interface {
	// Matte returns an alpha mask the same size as img (255 = person).
	Matte(ctx context.Context, img image.Image, raw []byte, f face.Result) (*image.Alpha, error)
}
