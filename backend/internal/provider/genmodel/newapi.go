package genmodel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NewAPI 使用兼容 OpenAI 的图像编辑接口，始终携带原图，不降级为文生图。
type NewAPI struct {
	key, model, base string
	client, download *http.Client
	price            int
}

func NewNewAPI(key, model, base string, price int) *NewAPI {
	base = strings.TrimRight(base, "/")
	if !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !ip.IP.IsGlobalUnicast() || ip.IP.IsPrivate() || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() {
				return nil, fmt.Errorf("图片地址不是公网地址")
			}
		}
		for _, ip := range ips {
			c, e := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if e == nil {
				return c, nil
			}
		}
		return nil, fmt.Errorf("图片下载连接失败")
	}
	return &NewAPI{key: key, model: model, base: base, price: price,
		client: &http.Client{Timeout: 240 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		download: &http.Client{Timeout: 60 * time.Second, Transport: transport, CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= 3 || r.URL.Scheme != "https" || r.URL.User != nil {
				return fmt.Errorf("不允许的图片跳转")
			}
			return nil
		}},
	}
}
func (n *NewAPI) Name() string { return "newapi" }
func (n *NewAPI) Capabilities() Caps {
	return Caps{Img2Img: true, Edit: true, Reference: true, MaskEdit: true, MaxSide: 1536, ReturnsAlpha: true}
}
func (n *NewAPI) Run(ctx context.Context, r Request) (Result, error) {
	if !n.Capabilities().Supports(r.Mode) || len(r.Source) == 0 {
		return Result{}, fmt.Errorf("生图需要有效模式及原图")
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	addImage := func(field string, data []byte) error {
		cfg, format, e := image.DecodeConfig(bytes.NewReader(data))
		if e != nil || cfg.Width*cfg.Height > 40_000_000 {
			return fmt.Errorf("输入图片无效或过大")
		}
		part, e := w.CreateFormFile(field, field+"."+format)
		if e != nil {
			return e
		}
		_, e = part.Write(data)
		return e
	}
	field := "image"
	if len(r.References) > 0 {
		field = "image[]"
	}
	if err := addImage(field, r.Source); err != nil {
		return Result{}, err
	}
	for _, ref := range r.References {
		if err := addImage(field, ref); err != nil {
			return Result{}, err
		}
	}
	if len(r.Mask) > 0 {
		if err := addImage("mask", r.Mask); err != nil {
			return Result{}, err
		}
	}
	model := r.Model
	if model == "" {
		model = n.model
	}
	size := "1024x1024"
	if r.Height > r.Width {
		size = "1024x1536"
	} else if r.Width > r.Height {
		size = "1536x1024"
	}
	prompt := "保留原图人物的身份、五官与脸部特征。" + r.Prompt
	if r.NegativePrompt != "" {
		prompt += "。避免：" + r.NegativePrompt
	}
	fields := map[string]string{"model": model, "prompt": prompt, "size": size, "n": "1", "quality": "high", "response_format": "b64_json"}
	for k, v := range r.Extra {
		s, ok := v.(string)
		if !ok {
			return Result{}, fmt.Errorf("生图参数 %s 必须为字符串", k)
		}
		switch k {
		case "quality":
			if s != "auto" && s != "low" && s != "medium" && s != "high" {
				return Result{}, fmt.Errorf("quality 无效")
			}
		case "background":
			if s != "auto" && s != "opaque" && s != "transparent" {
				return Result{}, fmt.Errorf("background 无效")
			}
		default:
			return Result{}, fmt.Errorf("不支持的生图参数 %s", k)
		}
		fields[k] = s
	}
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return Result{}, err
		}
	}
	if err := w.Close(); err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.base+"/images/edits", &body)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+n.key)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := n.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("生图请求未完成，请查询任务结果后重试")
	}
	defer resp.Body.Close()
	// 超时、429 与 5xx 不自动重发付费请求，以免结果未知时重复计费。
	if resp.StatusCode != 200 {
		return Result{}, fmt.Errorf("生图服务返回 HTTP %d", resp.StatusCode)
	}
	raw, err := readLimited(resp.Body, 48<<20)
	if err != nil {
		return Result{}, err
	}
	var out struct {
		Data []struct {
			B64 string `json:"b64_json"`
			URL string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &out) != nil || len(out.Data) != 1 {
		return Result{}, fmt.Errorf("生图响应缺少图片")
	}
	var data []byte
	if out.Data[0].B64 != "" {
		data, err = base64.StdEncoding.DecodeString(out.Data[0].B64)
	} else {
		u, e := url.Parse(out.Data[0].URL)
		if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return Result{}, fmt.Errorf("图片下载地址无效")
		}
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if e != nil {
			return Result{}, e
		}
		res, e := n.download.Do(req)
		if e != nil {
			return Result{}, fmt.Errorf("图片下载失败")
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return Result{}, fmt.Errorf("图片下载 HTTP %d", res.StatusCode)
		}
		data, err = readLimited(res.Body, 32<<20)
	}
	if err != nil {
		return Result{}, fmt.Errorf("图片响应读取失败")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width*cfg.Height > 40_000_000 {
		return Result{}, fmt.Errorf("服务返回的图片无效或过大")
	}
	return Result{Image: data, Format: format, CostCents: n.price, ProviderRef: resp.Header.Get("X-Request-Id"), HasAlpha: format == "png"}, nil
}
func readLimited(r io.Reader, limit int64) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, limit+1))
	if e == nil && int64(len(b)) > limit {
		e = fmt.Errorf("响应超过大小限制")
	}
	return b, e
}
