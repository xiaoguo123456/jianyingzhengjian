// Package inspect 在上传时用多模态模型检查照片能否用于生成人像，取代人脸检测服务。
package inspect

import "context"

// Report 是一次质检的结构化结果。Issues 只包含 Allowed 中的取值。
type Report struct {
	Faces  int
	Gender string // male | female | ""
	Issues []string
}

// Allowed 是模型可以返回的问题代码，与客户端 PHOTO_REASON_COPY 保持一致。
var Allowed = map[string]bool{
	"face_too_small": true,
	"blurry":         true,
	"too_dark":       true,
	"occluded":       true,
	"not_photo":      true,
}

type Inspector interface {
	Inspect(ctx context.Context, jpeg []byte) (Report, error)
}

// Mock 假定照片合格，仅用于开发和 CI。
type Mock struct{}

func (Mock) Inspect(context.Context, []byte) (Report, error) { return Report{Faces: 1}, nil }
