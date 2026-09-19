// Package local is the pure-Go imaging engine: decode, resize and crop, export, label.
package local

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/disintegration/imaging"
)

// Decode reads JPEG/PNG (auto-orienting by EXIF) and returns the format name.
func Decode(raw []byte) (image.Image, string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, "", fmt.Errorf("decode config: %w", err)
	}
	img, err := imaging.Decode(bytes.NewReader(raw), imaging.AutoOrientation(true))
	if err != nil {
		return nil, "", fmt.Errorf("decode: %w", err)
	}
	return img, format, nil
}

// Downscale limits the long side, returning the input unchanged when small enough.
func Downscale(img image.Image, maxSide int) image.Image {
	b := img.Bounds()
	if b.Dx() <= maxSide && b.Dy() <= maxSide {
		return img
	}
	if b.Dx() >= b.Dy() {
		return imaging.Resize(img, maxSide, 0, imaging.Lanczos)
	}
	return imaging.Resize(img, 0, maxSide, imaging.Lanczos)
}

func ToNRGBA(img image.Image) *image.NRGBA { return imaging.Clone(img) }

// Fill crops/resizes to exactly w×h keeping the centre.
func Fill(img image.Image, w, h int) *image.NRGBA {
	return imaging.Fill(img, w, h, imaging.Center, imaging.Lanczos)
}

func EncodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Thumb returns a JPEG thumbnail with the long side limited to maxSide.
func Thumb(img image.Image, maxSide int) ([]byte, error) {
	return EncodeJPEG(Downscale(img, maxSide), 82)
}
