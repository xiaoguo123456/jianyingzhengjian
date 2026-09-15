package genmodel

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func testPNG() []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 4, 4)))
	return b.Bytes()
}
func TestNewAPIEdit(t *testing.T) {
	src := testPNG()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" || r.Header.Get("Authorization") != "Bearer test" {
			t.Error("编辑路径或认证不正确")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			return
		}
		f, _, err := r.FormFile("image")
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		data, _ := io.ReadAll(f)
		if !bytes.Equal(data, src) {
			t.Error("原图被遗漏或修改")
		}
		if r.FormValue("size") != "1024x1536" || r.FormValue("model") != "gpt-image-2.5" {
			t.Error("模型或尺寸不正确")
		}
		fmt.Fprintf(w, `{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString(src))
	}))
	defer srv.Close()
	p := NewNewAPI("test", "gpt-image-2.5", srv.URL, 0)
	out, err := p.Run(context.Background(), Request{Mode: ModeImg2Img, Source: src, Width: 1200, Height: 1600})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Image, src) || out.Format != "png" {
		t.Fatal("输出解码错误")
	}
}
func TestNewAPIErrors(t *testing.T) {
	for _, body := range []string{`{"data":[]}`, `{"data":[{"url":"http://127.0.0.1/private"}]}`, `{"data":[{"b64_json":"bm90LWFuLWltYWdl"}]}`} {
		t.Run(body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer srv.Close()
			p := NewNewAPI("test", "m", srv.URL, 0)
			if _, err := p.Run(context.Background(), Request{Mode: ModeEdit, Source: testPNG()}); err == nil {
				t.Fatal("错误响应不能成功")
			}
		})
	}
}
func TestNewAPILive(t *testing.T) {
	key := os.Getenv("NEWAPI_LIVE_KEY")
	if key == "" {
		t.Skip("需要显式提供付费联调凭据")
	}
	src, err := os.ReadFile("../../../seed/assets/pro_interview.jpg")
	if err != nil {
		t.Fatal(err)
	}
	p := NewNewAPI(key, "gpt-image-2.5", "https://www.ggwk1.online/v1", 0)
	out, err := p.Run(context.Background(), Request{Mode: ModeEdit, Source: src, Prompt: "保持人物身份，将背景换为浅灰色专业摄影棚，深蓝西装，职业形象照。", Width: 1200, Height: 1600})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(os.TempDir()+"/yingji-newapi-live."+out.Format, out.Image, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("真实图像编辑成功，格式 %s，大小 %d", out.Format, len(out.Image))
}
