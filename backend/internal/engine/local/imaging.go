// Package local is the pure-Go imaging engine: decode, crop, composite, export, label.
package local

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"strings"

	"github.com/disintegration/imaging"

	"yingji/backend/internal/domain"
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

func Resize(img image.Image, w, h int) *image.NRGBA {
	return imaging.Resize(img, w, h, imaging.Lanczos)
}

// Fill crops/resizes to exactly w×h keeping the centre.
func Fill(img image.Image, w, h int) *image.NRGBA {
	return imaging.Fill(img, w, h, imaging.Center, imaging.Lanczos)
}

// DefaultCropRule per docs/GENERATION_PIPELINE.md §7.1.
var DefaultCropRule = domain.CropRule{HeadRatio: 0.62, TopMargin: 0.10}

// CropSpec crops the source around the detected face to the spec's aspect ratio
// and resizes to wpx×hpx. Out-of-frame areas are edge-extended.
func CropSpec(img image.Image, face image.Rectangle, wpx, hpx int, rule domain.CropRule) (*image.NRGBA, image.Rectangle) {
	if rule.HeadRatio <= 0 {
		rule.HeadRatio = DefaultCropRule.HeadRatio
	}
	if rule.TopMargin <= 0 {
		rule.TopMargin = DefaultCropRule.TopMargin
	}
	fh := float64(face.Dy())
	if fh <= 0 {
		fh = float64(img.Bounds().Dy()) * 0.3
	}
	estHead := 1.45 * fh
	scale := (float64(hpx) * rule.HeadRatio) / estHead // output px per source px
	headTop := float64(face.Min.Y) - 0.30*fh
	cropH := float64(hpx) / scale
	cropW := float64(wpx) / scale
	cropTop := headTop - rule.TopMargin*cropH
	cx := float64(face.Min.X+face.Max.X) / 2
	rect := image.Rect(int(math.Round(cx-cropW/2)), int(math.Round(cropTop)), int(math.Round(cx+cropW/2)), int(math.Round(cropTop+cropH)))
	canvas := extractExtended(img, rect)
	return imaging.Resize(canvas, wpx, hpx, imaging.Lanczos), rect
}

// extractExtended copies rect from img, extending edge pixels where rect leaves the image.
func extractExtended(img image.Image, rect image.Rectangle) *image.NRGBA {
	src := imaging.Clone(img)
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		sy := clamp(rect.Min.Y+y, b.Min.Y, b.Max.Y-1)
		for x := 0; x < rect.Dx(); x++ {
			sx := clamp(rect.Min.X+x, b.Min.X, b.Max.X-1)
			i := src.PixOffset(sx, sy)
			o := out.PixOffset(x, y)
			copy(out.Pix[o:o+4], src.Pix[i:i+4])
		}
	}
	return out
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// CompositeSolid places img over a solid colour using alpha (same size as img).
func CompositeSolid(img image.Image, alpha *image.Alpha, hex string) (*image.NRGBA, error) {
	bg, err := ParseHex(hex)
	if err != nil {
		return nil, err
	}
	src := imaging.Clone(img)
	b := src.Bounds()
	out := image.NewNRGBA(b)
	draw.Draw(out, b, &image.Uniform{bg}, image.Point{}, draw.Src)
	fg := image.NewNRGBA(b)
	copy(fg.Pix, src.Pix)
	if alpha != nil {
		ab := alpha.Bounds()
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				ax := ab.Min.X + x*ab.Dx()/b.Dx()
				ay := ab.Min.Y + y*ab.Dy()/b.Dy()
				a := alpha.AlphaAt(ax, ay).A
				fg.Pix[fg.PixOffset(b.Min.X+x, b.Min.Y+y)+3] = a
			}
		}
	}
	draw.Draw(out, b, fg, b.Min, draw.Over)
	return out, nil
}

// SquareCropFace returns a 1:1 crop centred slightly above the face centre.
func SquareCropFace(img image.Image, face image.Rectangle, side int) *image.NRGBA {
	b := img.Bounds()
	size := int(math.Min(float64(b.Dx()), float64(b.Dy())))
	cx := (face.Min.X + face.Max.X) / 2
	cy := (face.Min.Y+face.Max.Y)/2 + face.Dy()/4
	if face.Empty() {
		cx, cy = b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2
	}
	rect := image.Rect(cx-size/2, cy-size/2, cx+size/2, cy+size/2)
	rect = rect.Add(image.Point{X: clamp(0, b.Min.X-rect.Min.X, b.Max.X-rect.Max.X), Y: clamp(0, b.Min.Y-rect.Min.Y, b.Max.Y-rect.Max.Y)})
	crop := imaging.Crop(img, rect)
	if side > 0 {
		return imaging.Resize(crop, side, side, imaging.Lanczos)
	}
	return crop
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

func EncodeAlphaPNG(a *image.Alpha) ([]byte, error) {
	g := image.NewGray(a.Bounds())
	copy(g.Pix, a.Pix)
	return EncodePNG(g)
}

func DecodeAlphaPNG(data []byte) (*image.Alpha, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	a := image.NewAlpha(b)
	g := image.NewGray(b)
	draw.Draw(g, b, img, b.Min, draw.Src)
	copy(a.Pix, g.Pix)
	return a, nil
}

func ParseHex(s string) (color.NRGBA, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return color.NRGBA{}, errors.New("invalid colour")
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return color.NRGBA{}, errors.New("invalid colour")
	}
	return color.NRGBA{R: r, G: g, B: b, A: 255}, nil
}

// BlurScore is the variance of a Laplacian over a downscaled grey copy (higher = sharper).
func BlurScore(img image.Image) float64 {
	small := imaging.Grayscale(Downscale(img, 512))
	b := small.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 3 || h < 3 {
		return 0
	}
	var sum, sumSq float64
	n := 0
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			c := float64(small.Pix[small.PixOffset(x, y)])
			l := 4*c - float64(small.Pix[small.PixOffset(x-1, y)]) - float64(small.Pix[small.PixOffset(x+1, y)]) -
				float64(small.Pix[small.PixOffset(x, y-1)]) - float64(small.Pix[small.PixOffset(x, y+1)])
			sum += l
			sumSq += l * l
			n++
		}
	}
	mean := sum / float64(n)
	return sumSq/float64(n) - mean*mean
}

// MeanLuma returns the mean luminance (0..255) inside rect.
func MeanLuma(img image.Image, rect image.Rectangle) float64 {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		rect = img.Bounds()
	}
	var sum float64
	n := 0
	step := 1 + rect.Dx()/200
	for y := rect.Min.Y; y < rect.Max.Y; y += step {
		for x := rect.Min.X; x < rect.Max.X; x += step {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
