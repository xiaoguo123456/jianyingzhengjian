package genmodel

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

	"yingji/backend/internal/pkg/apperr"
)

// Volcengine calls the Ark image generation API (Doubao Seedream). The request
// shape follows the images/generations endpoint; confirm field names against the
// current Ark documentation before production use.
type Volcengine struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
	price   int // cents per image, from provider_prices
}

func NewVolcengine(apiKey, model, baseURL string, priceCents int) *Volcengine {
	return &Volcengine{apiKey: apiKey, model: model, baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{Timeout: 120 * time.Second}, price: priceCents}
}

func (v *Volcengine) Name() string { return "volcengine" }

func (v *Volcengine) Capabilities() Caps {
	return Caps{Img2Img: true, Reference: true, Edit: true, MaskEdit: false, MaxSide: 4096}
}

func (v *Volcengine) Run(ctx context.Context, req Request) (Result, error) {
	model := req.Model
	if model == "" {
		model = v.model
	}
	prompt := req.Prompt
	if req.NegativePrompt != "" {
		prompt += " --no " + req.NegativePrompt
	}
	body := map[string]any{
		"model":           model,
		"prompt":          prompt,
		"image":           "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(req.Source),
		"response_format": "b64_json",
		"watermark":       false,
	}
	if req.Width > 0 && req.Height > 0 {
		body["size"] = fmt.Sprintf("%dx%d", req.Width, req.Height)
	}
	if req.Seed > 0 {
		body["seed"] = req.Seed
	}
	if req.Mode == ModeReference && len(req.References) > 0 {
		imgs := []string{"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(req.Source)}
		for _, r := range req.References {
			imgs = append(imgs, "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString(r))
		}
		body["image"] = imgs
	}
	for k, val := range req.Extra {
		body[k] = val
	}
	raw, _ := json.Marshal(body)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/images/generations", bytes.NewReader(raw))
	httpReq.Header.Set("Authorization", "Bearer "+v.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := v.http.Do(httpReq)
	if err != nil {
		return Result{}, apperr.Transient("PROVIDER_ERROR", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 429 || resp.StatusCode >= 500 {
		return Result{}, apperr.Transient("PROVIDER_ERROR", fmt.Errorf("volcengine http %d: %s", resp.StatusCode, truncate(data)))
	}
	if resp.StatusCode >= 400 {
		if strings.Contains(string(data), "sensitive") || strings.Contains(string(data), "content") {
			return Result{}, apperr.New("CONTENT_REJECTED", 422, "content rejected").WithCause(fmt.Errorf("%s", truncate(data)))
		}
		return Result{}, fmt.Errorf("volcengine http %d: %s", resp.StatusCode, truncate(data))
	}
	var out struct {
		Data []struct {
			B64  string `json:"b64_json"`
			URL  string `json:"url"`
			Size string `json:"size"`
		} `json:"data"`
		Usage struct {
			GeneratedImages int `json:"generated_images"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &out); err != nil || len(out.Data) == 0 {
		return Result{}, fmt.Errorf("volcengine: bad response: %s", truncate(data))
	}
	var img []byte
	if out.Data[0].B64 != "" {
		img, err = base64.StdEncoding.DecodeString(out.Data[0].B64)
		if err != nil {
			return Result{}, err
		}
	} else if out.Data[0].URL != "" {
		r, err := v.http.Get(out.Data[0].URL)
		if err != nil {
			return Result{}, apperr.Transient("PROVIDER_ERROR", err)
		}
		defer r.Body.Close()
		img, _ = io.ReadAll(r.Body)
	}
	return Result{Image: img, Format: "jpeg", CostCents: v.price, ProviderRef: fmt.Sprintf("ark-%d", time.Now().UnixNano()), Seed: req.Seed}, nil
}

func truncate(b []byte) string {
	if len(b) > 300 {
		return string(b[:300])
	}
	return string(b)
}
