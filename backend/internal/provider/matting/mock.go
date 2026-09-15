package matting

import (
	"context"
	"image"
	"math"

	"yingji/backend/internal/provider/face"
)

// Mock segments the subject by keying the background colour and flood-filling inward
// from the image border. It is not a segmentation model: it works on ID-photo-style
// shots with a reasonably even backdrop and degrades on busy backgrounds. Production
// uses matting.Tencent. Flood-filling from the border (rather than thresholding every
// pixel) keeps background-coloured areas inside the subject, such as a white shirt.
type Mock struct {
	// Tolerance is the squared RGB distance from the background reference that still
	// counts as background. Zero selects a sensible default.
	Tolerance float64
	// StepTolerance is the largest squared colour step the fill may cross between
	// neighbouring pixels. It stops the region growing at a visible edge, so a
	// near-white collar is not swallowed by a light background. Zero selects a default.
	StepTolerance float64
}

func (m Mock) Matte(ctx context.Context, img image.Image, raw []byte, f face.Result) (*image.Alpha, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	a := image.NewAlpha(b)
	if w == 0 || h == 0 {
		return a, nil
	}
	tol := m.Tolerance
	if tol <= 0 {
		tol = 2200 // ~27 per channel
	}
	stepTol := m.StepTolerance
	if stepTol <= 0 {
		stepTol = 400 // ~11 per channel between neighbours
	}

	px := make([][3]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			px[y*w+x] = [3]float64{float64(r >> 8), float64(g >> 8), float64(bl >> 8)}
		}
	}

	// Background reference: the median of the top corners, which are background in a portrait.
	ref := medianCorner(px, w, h)

	// Flood fill from the border through pixels close to the reference colour.
	bg := make([]bool, w*h)
	queue := make([]int, 0, w*h/4)
	// seed accepts a border pixel; grow accepts a neighbour only if it is both close to
	// the background reference and a small colour step from the pixel we came from.
	seed := func(x, y int) {
		i := y*w + x
		if bg[i] || dist2(px[i], ref) > tol {
			return
		}
		bg[i] = true
		queue = append(queue, i)
	}
	grow := func(from, x, y int) {
		i := y*w + x
		if bg[i] || dist2(px[i], ref) > tol || dist2(px[i], px[from]) > stepTol {
			return
		}
		bg[i] = true
		queue = append(queue, i)
	}
	// Seed from the top and sides only: in a portrait the subject's body always meets the
	// bottom edge, so seeding there leaks the fill into clothing.
	for x := 0; x < w; x++ {
		seed(x, 0)
	}
	for y := 0; y < h; y++ {
		seed(0, y)
		seed(w-1, y)
	}
	for len(queue) > 0 {
		i := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		x, y := i%w, i/w
		if x > 0 {
			grow(i, x-1, y)
		}
		if x < w-1 {
			grow(i, x+1, y)
		}
		if y > 0 {
			grow(i, x, y-1)
		}
		if y < h-1 {
			grow(i, x, y+1)
		}
	}

	// Subject = everything the fill did not reach.
	hard := make([]float64, w*h)
	subject := 0
	for i := range bg {
		if !bg[i] {
			hard[i] = 1
			subject++
		}
	}
	// If keying failed (almost everything kept or removed), keep the whole frame rather
	// than returning a mask that would mangle the photo.
	if ratio := float64(subject) / float64(w*h); ratio > 0.97 || ratio < 0.05 {
		for i := range a.Pix {
			a.Pix[i] = 255
		}
		return a, nil
	}

	// Feather the boundary so hair edges do not look cut out.
	radius := int(math.Max(1, float64(w)/200))
	soft := boxBlur(hard, w, h, radius)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := soft[y*w+x]
			a.Pix[y*a.Stride+x] = uint8(math.Max(0, math.Min(1, v)) * 255)
		}
	}
	return a, nil
}

func dist2(a, b [3]float64) float64 {
	dr, dg, db := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return dr*dr + dg*dg + db*db
}

// medianCorner averages the two top corners, which are background in a portrait.
func medianCorner(px [][3]float64, w, h int) [3]float64 {
	n := int(math.Max(2, float64(w)/20))
	var sum [3]float64
	count := 0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			for _, i := range []int{y*w + x, y*w + (w - 1 - x)} {
				p := px[i]
				sum[0] += p[0]
				sum[1] += p[1]
				sum[2] += p[2]
				count++
			}
		}
	}
	if count == 0 {
		return [3]float64{255, 255, 255}
	}
	return [3]float64{sum[0] / float64(count), sum[1] / float64(count), sum[2] / float64(count)}
}

// boxBlur runs a separable box blur over a single-channel float image.
func boxBlur(src []float64, w, h, r int) []float64 {
	if r < 1 {
		return src
	}
	tmp := make([]float64, len(src))
	out := make([]float64, len(src))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum float64
			var n int
			for dx := -r; dx <= r; dx++ {
				if xx := x + dx; xx >= 0 && xx < w {
					sum += src[y*w+xx]
					n++
				}
			}
			tmp[y*w+x] = sum / float64(n)
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum float64
			var n int
			for dy := -r; dy <= r; dy++ {
				if yy := y + dy; yy >= 0 && yy < h {
					sum += tmp[yy*w+x]
					n++
				}
			}
			out[y*w+x] = sum / float64(n)
		}
	}
	return out
}
