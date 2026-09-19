package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestNewAPIInspect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test" {
			t.Error("质检路径或认证不正确")
		}
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Content []struct {
					Type     string            `json:"type"`
					ImageURL map[string]string `json:"image_url"`
				} `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "vision-model" || len(req.Messages) != 1 || len(req.Messages[0].Content) != 2 ||
			!strings.HasPrefix(req.Messages[0].Content[1].ImageURL["url"], "data:image/jpeg;base64,") {
			t.Error("请求未携带模型或图片")
		}
		answer := "```json\n{\"faces\": 1, \"gender\": \"Female\", \"issues\": [\"blurry\", \"unknown\", \"blurry\"]}\n```"
		b, _ := json.Marshal(answer)
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%s}}]}`, b)
	}))
	defer srv.Close()
	got, err := NewNewAPI("test", "vision-model", srv.URL).Inspect(context.Background(), []byte("jpeg"))
	if err != nil {
		t.Fatal(err)
	}
	want := Report{Faces: 1, Gender: "female", Issues: []string{"blurry"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNewAPIInspectErrors(t *testing.T) {
	for _, body := range []string{`{"choices":[]}`, `{"choices":[{"message":{"content":"无法判断"}}]}`, `{"choices":[{"message":{"content":"{\"gender\":\"male\"}"}}]}`} {
		t.Run(body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer srv.Close()
			if _, err := NewNewAPI("test", "m", srv.URL).Inspect(context.Background(), []byte("jpeg")); err == nil {
				t.Fatal("无效质检结果未报错")
			}
		})
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer srv.Close()
	if _, err := NewNewAPI("test", "m", srv.URL).Inspect(context.Background(), []byte("jpeg")); err == nil {
		t.Fatal("HTTP 错误未报错")
	}
}
