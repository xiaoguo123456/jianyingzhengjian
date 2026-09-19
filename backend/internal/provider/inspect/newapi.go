package inspect

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const prompt = `你是人像照片生成服务的照片质检员，判断这张照片能否用来生成证件照或写真。
只输出一个 JSON 对象，不要输出任何其他文字：
{"faces": 照片中清晰可见的真人人脸数量（整数）, "gender": 主体人物性别 "male" 或 "female"，无法判断时为 "", "issues": [问题代码]}
issues 只能从下面取值，没有问题时为空数组：
face_too_small：人脸太小，面部高度不足画面高度的八分之一
blurry：面部模糊或失焦
too_dark：面部过暗或严重逆光
occluded：面部被墨镜、口罩、手或头发明显遮挡
not_photo：不是真人照片，例如卡通、绘画、屏幕翻拍或截图`

// NewAPI 通过兼容 OpenAI 的 /chat/completions 调用多模态模型。
type NewAPI struct {
	key, model, base string
	client           *http.Client
}

func NewNewAPI(key, model, base string) *NewAPI {
	base = strings.TrimRight(base, "/")
	if !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}
	return &NewAPI{key: key, model: model, base: base,
		client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (n *NewAPI) Inspect(ctx context.Context, jpeg []byte) (Report, error) {
	body, _ := json.Marshal(map[string]any{
		"model":       n.model,
		"temperature": 0,
		"messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": prompt},
			map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpeg)}},
		}}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Report{}, err
	}
	req.Header.Set("Authorization", "Bearer "+n.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return Report{}, fmt.Errorf("质检请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Report{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Report{}, fmt.Errorf("质检服务返回 HTTP %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &out) != nil || len(out.Choices) == 0 {
		return Report{}, fmt.Errorf("质检响应格式无效")
	}
	return parse(out.Choices[0].Message.Content)
}

// parse 取回答中的第一个 JSON 对象，兼容模型在 JSON 外加代码块或说明文字。
func parse(content string) (Report, error) {
	start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return Report{}, fmt.Errorf("质检结果不是 JSON")
	}
	var v struct {
		Faces  *int     `json:"faces"`
		Gender string   `json:"gender"`
		Issues []string `json:"issues"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &v); err != nil || v.Faces == nil || *v.Faces < 0 {
		return Report{}, fmt.Errorf("质检结果缺少人脸数量")
	}
	r := Report{Faces: *v.Faces}
	if g := strings.ToLower(strings.TrimSpace(v.Gender)); g == "male" || g == "female" {
		r.Gender = g
	}
	seen := map[string]bool{}
	for _, is := range v.Issues {
		if Allowed[is] && !seen[is] {
			seen[is] = true
			r.Issues = append(r.Issues, is)
		}
	}
	return r, nil
}
