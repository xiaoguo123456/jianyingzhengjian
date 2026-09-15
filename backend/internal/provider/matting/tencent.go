package matting

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"yingji/backend/internal/provider/face"
	"yingji/backend/internal/provider/tencentcloud"
)

// Tencent uses Body Analysis (bda) SegmentPortraitPic. Untested against the live API.
type Tencent struct {
	cli *tencentcloud.Client
}

func NewTencent(secretID, secretKey, region string) *Tencent {
	return &Tencent{cli: tencentcloud.New(secretID, secretKey, region, "bda", "2020-03-24")}
}

func (t *Tencent) Matte(ctx context.Context, img image.Image, raw []byte, f face.Result) (*image.Alpha, error) {
	var out struct {
		Response struct {
			ResultMask string
			Error      *struct{ Code, Message string }
		}
	}
	if err := t.cli.Call(ctx, "SegmentPortraitPic", map[string]any{"Image": base64.StdEncoding.EncodeToString(raw)}, &out); err != nil {
		return nil, err
	}
	if out.Response.Error != nil {
		return nil, fmt.Errorf("bda SegmentPortraitPic: %s %s", out.Response.Error.Code, out.Response.Error.Message)
	}
	maskPNG, err := base64.StdEncoding.DecodeString(out.Response.ResultMask)
	if err != nil {
		return nil, err
	}
	m, err := png.Decode(bytes.NewReader(maskPNG))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	a := image.NewAlpha(b)
	gray := image.NewGray(m.Bounds())
	draw.Draw(gray, gray.Bounds(), m, m.Bounds().Min, draw.Src)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			mx := x * gray.Bounds().Dx() / b.Dx()
			my := y * gray.Bounds().Dy() / b.Dy()
			a.Pix[y*a.Stride+x] = gray.Pix[my*gray.Stride+mx]
		}
	}
	return a, nil
}
