// Package tencentcloud is a minimal TC3-HMAC-SHA256 signed JSON client, enough for
// the iai (face) and bda (portrait segmentation) APIs without the full SDK.
package tencentcloud

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	secretID  string
	secretKey string
	region    string
	service   string
	version   string
	http      *http.Client
}

func New(secretID, secretKey, region, service, version string) *Client {
	return &Client{secretID: secretID, secretKey: secretKey, region: region, service: service, version: version,
		http: &http.Client{Timeout: 30 * time.Second}}
}

func hmacSHA256(key []byte, msg string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	return h.Sum(nil)
}

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// Call performs a signed POST for the named action.
func (c *Client) Call(ctx context.Context, action string, params any, out any) error {
	host := c.service + ".tencentcloudapi.com"
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	now := time.Now()
	ts := strconv.FormatInt(now.Unix(), 10)
	date := now.UTC().Format("2006-01-02")

	canonicalHeaders := "content-type:application/json\nhost:" + host + "\n"
	signedHeaders := "content-type;host"
	canonicalRequest := strings.Join([]string{"POST", "/", "", canonicalHeaders, signedHeaders, sha256Hex(payload)}, "\n")
	credentialScope := date + "/" + c.service + "/tc3_request"
	stringToSign := strings.Join([]string{"TC3-HMAC-SHA256", ts, credentialScope, sha256Hex([]byte(canonicalRequest))}, "\n")

	secretDate := hmacSHA256([]byte("TC3"+c.secretKey), date)
	secretService := hmacSHA256(secretDate, c.service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))
	authorization := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", c.secretID, credentialScope, signedHeaders, signature)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+host, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", ts)
	req.Header.Set("X-TC-Version", c.version)
	if c.region != "" {
		req.Header.Set("X-TC-Region", c.region)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("tencentcloud %s: http %d", action, resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}
