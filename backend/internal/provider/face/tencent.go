package face

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"

	"yingji/backend/internal/provider/tencentcloud"
)

// Tencent uses Tencent Cloud Face Recognition (iai) DetectFace / CompareFace.
// Untested against the live API; parameter names follow the 2020-03-03 version.
type Tencent struct {
	cli *tencentcloud.Client
}

func NewTencent(secretID, secretKey, region string) *Tencent {
	return &Tencent{cli: tencentcloud.New(secretID, secretKey, region, "iai", "2020-03-03")}
}

func (t *Tencent) Detect(ctx context.Context, img image.Image, raw []byte) (Result, error) {
	var out struct {
		Response struct {
			FaceInfos []struct {
				X, Y, Width, Height int
				FaceAttributesInfo  struct {
					Gender int
					Mask   int
					Glass  bool
				}
				FaceQualityInfo struct {
					Score int
				}
			}
			Error *struct{ Code, Message string }
		}
	}
	req := map[string]any{"Image": base64.StdEncoding.EncodeToString(raw), "MaxFaceNum": 3, "NeedFaceAttributes": 1, "NeedQualityDetection": 1}
	if err := t.cli.Call(ctx, "DetectFace", req, &out); err != nil {
		return Result{}, err
	}
	if out.Response.Error != nil {
		if out.Response.Error.Code == "FailedOperation.NoFaceDetected" {
			return Result{Faces: 0}, nil
		}
		return Result{}, fmt.Errorf("iai DetectFace: %s %s", out.Response.Error.Code, out.Response.Error.Message)
	}
	if len(out.Response.FaceInfos) == 0 {
		return Result{Faces: 0}, nil
	}
	f := out.Response.FaceInfos[0]
	gender := ""
	if f.FaceAttributesInfo.Gender >= 50 {
		gender = "male"
	} else if f.FaceAttributesInfo.Gender > 0 {
		gender = "female"
	}
	return Result{
		Faces:    len(out.Response.FaceInfos),
		Box:      image.Rect(f.X, f.Y, f.X+f.Width, f.Y+f.Height),
		Quality:  float64(f.FaceQualityInfo.Score) / 100,
		Gender:   gender,
		Occluded: f.FaceAttributesInfo.Mask > 0,
	}, nil
}

func (t *Tencent) Compare(ctx context.Context, a, b []byte) (float64, error) {
	var out struct {
		Response struct {
			Score float64
			Error *struct{ Code, Message string }
		}
	}
	req := map[string]any{"ImageA": base64.StdEncoding.EncodeToString(a), "ImageB": base64.StdEncoding.EncodeToString(b)}
	if err := t.cli.Call(ctx, "CompareFace", req, &out); err != nil {
		return 0, err
	}
	if out.Response.Error != nil {
		return 0, fmt.Errorf("iai CompareFace: %s %s", out.Response.Error.Code, out.Response.Error.Message)
	}
	return out.Response.Score / 100, nil
}
