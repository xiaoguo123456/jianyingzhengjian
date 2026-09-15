// Package wechat wraps the WeChat Mini Program server APIs used by the product.
package wechat

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	appID  string
	secret string
	http   *http.Client
	rdb    *redis.Client
	dev    bool
}

// New returns a client. When appID is empty and dev is true, Code2Session returns a
// deterministic dev identity so the H5 preview can log in without WeChat.
func New(appID, secret string, rdb *redis.Client, dev bool) *Client {
	return &Client{appID: appID, secret: secret, http: &http.Client{Timeout: 10 * time.Second}, rdb: rdb, dev: dev}
}

func (c *Client) Configured() bool { return c.appID != "" && c.secret != "" }
func (c *Client) AppID() string    { return c.appID }

type Session struct {
	OpenID     string
	UnionID    string
	SessionKey string
}

type wxErr struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (c *Client) Code2Session(ctx context.Context, code string) (Session, error) {
	if !c.Configured() {
		if c.dev {
			return Session{OpenID: "dev-openid-" + code}, nil
		}
		return Session{}, errors.New("wechat: not configured")
	}
	q := url.Values{"appid": {c.appID}, "secret": {c.secret}, "js_code": {code}, "grant_type": {"authorization_code"}}
	var out struct {
		wxErr
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
	}
	if err := c.getJSON(ctx, "https://api.weixin.qq.com/sns/jscode2session?"+q.Encode(), &out); err != nil {
		return Session{}, err
	}
	if out.ErrCode != 0 {
		return Session{}, fmt.Errorf("wechat code2session: %d %s", out.ErrCode, out.ErrMsg)
	}
	return Session{OpenID: out.OpenID, UnionID: out.UnionID, SessionKey: out.SessionKey}, nil
}

// AccessToken returns the app access token, cached in Redis.
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	if !c.Configured() {
		return "", errors.New("wechat: not configured")
	}
	const key = "wx:access_token"
	if c.rdb != nil {
		if v, err := c.rdb.Get(ctx, key).Result(); err == nil && v != "" {
			return v, nil
		}
	}
	q := url.Values{"grant_type": {"client_credential"}, "appid": {c.appID}, "secret": {c.secret}}
	var out struct {
		wxErr
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := c.getJSON(ctx, "https://api.weixin.qq.com/cgi-bin/token?"+q.Encode(), &out); err != nil {
		return "", err
	}
	if out.ErrCode != 0 {
		return "", fmt.Errorf("wechat token: %d %s", out.ErrCode, out.ErrMsg)
	}
	if c.rdb != nil {
		ttl := time.Duration(out.ExpiresIn-300) * time.Second
		if ttl < time.Minute {
			ttl = time.Minute
		}
		_ = c.rdb.Set(ctx, key, out.AccessToken, ttl).Err()
	}
	return out.AccessToken, nil
}

// SendSubscribeMessage sends a subscribe message; data values are {"value": "..."}.
func (c *Client) SendSubscribeMessage(ctx context.Context, openid, templateID, page string, data map[string]string) error {
	tok, err := c.AccessToken(ctx)
	if err != nil {
		return err
	}
	payload := map[string]any{"touser": openid, "template_id": templateID, "page": page, "data": map[string]map[string]string{}}
	d := payload["data"].(map[string]map[string]string)
	for k, v := range data {
		d[k] = map[string]string{"value": v}
	}
	var out wxErr
	if err := c.postJSON(ctx, "https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token="+tok, payload, &out); err != nil {
		return err
	}
	if out.ErrCode != 0 {
		return fmt.Errorf("wechat subscribe send: %d %s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

// MediaCheckAsync submits an image URL for content moderation (docs/DECISIONS.md D-15).
func (c *Client) MediaCheckAsync(ctx context.Context, mediaURL, openid string, scene int) (string, error) {
	tok, err := c.AccessToken(ctx)
	if err != nil {
		return "", err
	}
	payload := map[string]any{"media_url": mediaURL, "media_type": 2, "version": 2, "scene": scene, "openid": openid}
	var out struct {
		wxErr
		TraceID string `json:"trace_id"`
	}
	if err := c.postJSON(ctx, "https://api.weixin.qq.com/wxa/media_check_async?access_token="+tok, payload, &out); err != nil {
		return "", err
	}
	if out.ErrCode != 0 {
		return "", fmt.Errorf("wechat media check: %d %s", out.ErrCode, out.ErrMsg)
	}
	return out.TraceID, nil
}

// GetUnlimitedQRCode returns a mini program code PNG for the scene (docs/SHARING.md §5).
func (c *Client) GetUnlimitedQRCode(ctx context.Context, scene, page string, width int) ([]byte, error) {
	tok, err := c.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{"scene": scene, "page": page, "width": width, "check_path": false}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token="+tok, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		var e wxErr
		_ = json.Unmarshal(data, &e)
		return nil, fmt.Errorf("wechat wxacode: %d %s", e.ErrCode, e.ErrMsg)
	}
	return data, nil
}

// VerifyAdCallback checks the rewarded-video server callback signature.
// WeChat signs with SHA1 over "secret:" + a fixed parameter order; confirm the exact
// scheme in the traffic-master console before enabling (docs/API.md §11).
func VerifyAdCallback(q url.Values, secret string) bool {
	if secret == "" {
		return false
	}
	parts := []string{secret, q.Get("user_id"), q.Get("trans_id"), q.Get("trace_id"), q.Get("extra")}
	h := sha1.Sum([]byte(strings.Join(parts, ":")))
	return strings.EqualFold(hex.EncodeToString(h[:]), q.Get("sign"))
}

func (c *Client) getJSON(ctx context.Context, u string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) postJSON(ctx context.Context, u string, in, out any) error {
	body, _ := json.Marshal(in)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}
