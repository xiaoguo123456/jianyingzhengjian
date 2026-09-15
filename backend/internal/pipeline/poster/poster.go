// Package poster renders the 750×1334 share poster (docs/SHARING.md §5).
package poster

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/disintegration/imaging"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"yingji/backend/internal/engine/local"
)

const W, H = 750, 1334

type Input struct {
	Image    image.Image // the work
	Square   bool
	Title    string
	Subtitle string
	Brand    string
	QR       image.Image // mini program code, may be nil
	FontPath string
}

func Render(in Input) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, W, H))
	draw.Draw(out, out.Bounds(), &image.Uniform{color.NRGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)

	// image area
	areaH := 1000
	area := image.Rect(0, 0, W, areaH)
	draw.Draw(out, area, &image.Uniform{color.NRGBA{232, 241, 255, 255}}, image.Point{}, draw.Src)
	if in.Image != nil {
		var fitted *image.NRGBA
		if in.Square {
			fitted = imaging.Fit(in.Image, W-64, areaH-64, imaging.Lanczos)
		} else {
			fitted = imaging.Fill(in.Image, W, areaH, imaging.Center, imaging.Lanczos)
		}
		off := image.Point{X: (W - fitted.Bounds().Dx()) / 2, Y: (areaH - fitted.Bounds().Dy()) / 2}
		draw.Draw(out, fitted.Bounds().Add(off), fitted, image.Point{}, draw.Over)
		local.DrawBadge(out, in.FontPath)
	}

	// text
	if title := local.Face(in.FontPath, 40); title != nil {
		drawText(out, title, in.Title, 32, areaH+72, color.NRGBA{17, 24, 39, 255})
		if sub := local.Face(in.FontPath, 26); sub != nil {
			drawText(out, sub, in.Subtitle, 32, areaH+124, color.NRGBA{75, 85, 99, 255})
		}
		if brand := local.Face(in.FontPath, 24); brand != nil {
			drawText(out, brand, in.Brand, 32, H-56, color.NRGBA{47, 123, 246, 255})
		}
	}

	// mini program code bottom-right
	qrSize := 200
	if in.QR != nil {
		qr := imaging.Resize(in.QR, qrSize, qrSize, imaging.Lanczos)
		draw.Draw(out, qr.Bounds().Add(image.Point{X: W - qrSize - 32, Y: H - qrSize - 40}), qr, image.Point{}, draw.Over)
	} else {
		ph := image.Rect(W-qrSize-32, H-qrSize-40, W-32, H-40)
		draw.Draw(out, ph, &image.Uniform{color.NRGBA{230, 236, 245, 255}}, image.Point{}, draw.Src)
	}
	return out
}

func drawText(dst *image.NRGBA, face font.Face, s string, x, y int, c color.NRGBA) {
	d := &font.Drawer{Dst: dst, Src: &image.Uniform{c}, Face: face, Dot: fixed.P(x, y)}
	d.DrawString(s)
}
