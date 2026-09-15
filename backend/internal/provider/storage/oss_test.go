package storage

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOSSIsolationAndSigning(t *testing.T) {
	s, err := NewOSS("cn-beijing", "https://oss-cn-beijing-internal.aliyuncs.com", "", "test-bucket", "yingji/test", "test-id", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../prod/file", "/absolute", "a/../b", "a\\b", "a//b"} {
		if _, err := s.objectKey(key); err == nil {
			t.Errorf("未拒绝路径 %s", key)
		}
	}
	raw, err := s.SignedURL(context.Background(), "photos/a.jpg", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(raw)
	if strings.Contains(u.Host, "internal") || u.Path != "/yingji/test/photos/a.jpg" || u.Query().Get("x-oss-signature") == "" {
		t.Fatal("签名必须使用公网端点和环境目录")
	}
	if s.PublicURL("photos/a.jpg") != "" {
		t.Fatal("私有照片不能走公开素材方法")
	}
}
