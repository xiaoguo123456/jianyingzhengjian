// storagecheck 对当前环境执行一次小对象写入、读取、签名与删除验收。
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	"yingji/backend/internal/config"
	"yingji/backend/internal/provider/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	c, e := config.Load()
	if e != nil {
		return e
	}
	s, e := storage.NewOSS(c.OSSRegion, c.OSSEndpoint, c.OSSPublicEndpoint, c.OSSBucket, c.OSSPrefix, c.OSSAccessKeyID, c.OSSAccessKeySecret)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	key := fmt.Sprintf("checks/%d.txt", time.Now().UnixNano())
	payload := []byte("映己 OSS 隔离验收")
	if e = s.Put(ctx, key, bytes.NewReader(payload), int64(len(payload)), "text/plain"); e != nil {
		return e
	}
	defer s.Delete(context.Background(), key)
	r, e := s.Get(ctx, key)
	if e != nil {
		return e
	}
	b, e := io.ReadAll(r)
	r.Close()
	if e != nil || !bytes.Equal(b, payload) {
		return fmt.Errorf("OSS 读取不匹配")
	}
	u, e := s.SignedURL(ctx, key, time.Minute)
	if e != nil {
		return e
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	client := http.Client{Timeout: 20 * time.Second}
	res, e := client.Do(req)
	if e != nil {
		return fmt.Errorf("签名链接请求失败")
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("签名链接状态 %d", res.StatusCode)
	}
	req.URL.RawQuery = ""
	res, e = client.Do(req)
	if e != nil {
		return fmt.Errorf("私有权限校验失败")
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		return fmt.Errorf("未签名请求应为 403，实际 %d", res.StatusCode)
	}
	if e = s.Delete(ctx, key); e != nil {
		return e
	}
	fmt.Println("OSS 写入、读取、公网签名、私有权限与删除验收通过")
	return nil
}
