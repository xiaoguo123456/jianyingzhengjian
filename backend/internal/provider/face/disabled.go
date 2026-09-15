package face

import (
	"context"
	"fmt"
	"image"
)

// Disabled 在未配置生产视觉服务时明确拒绝处理，避免用模拟检测结果生成证件照。
type Disabled struct{}

func (Disabled) Detect(context.Context, image.Image, []byte) (Result, error) {
	return Result{}, fmt.Errorf("人脸检测服务尚未配置")
}
func (Disabled) Compare(context.Context, []byte, []byte) (float64, error) {
	return 0, fmt.Errorf("人脸比对服务尚未配置")
}
