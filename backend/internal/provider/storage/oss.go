package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// OSS 所有对象保持私有，通过公网端点签名，不修改共享 Bucket 的权限。
type OSS struct {
	client, signer *oss.Client
	bucket, prefix string
}

func NewOSS(region, endpoint, publicEndpoint, bucket, prefix, key, secret string) (*OSS, error) {
	if region == "" || bucket == "" || key == "" || secret == "" {
		return nil, fmt.Errorf("OSS 配置不完整")
	}
	prefix = strings.Trim(prefix, "/")
	if prefix == "" || strings.Contains(prefix, "..") || strings.Contains(prefix, "\\") {
		return nil, fmt.Errorf("OSS 必须设置独立目录前缀")
	}
	if endpoint == "" {
		endpoint = "https://oss-" + region + ".aliyuncs.com"
	}
	if publicEndpoint == "" {
		publicEndpoint = "https://oss-" + region + ".aliyuncs.com"
	}
	for _, s := range []string{endpoint, publicEndpoint} {
		u, e := url.Parse(s)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			return nil, fmt.Errorf("OSS 端点必须使用 HTTPS")
		}
	}
	provider := credentials.NewStaticCredentialsProvider(key, secret)
	makeClient := func(ep string) *oss.Client {
		return oss.NewClient(oss.LoadDefaultConfig().WithRegion(region).WithEndpoint(ep).WithCredentialsProvider(provider))
	}
	return &OSS{client: makeClient(endpoint), signer: makeClient(publicEndpoint), bucket: bucket, prefix: prefix + "/"}, nil
}
func (o *OSS) objectKey(key string) (string, error) {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "\\") {
		return "", fmt.Errorf("对象路径无效")
	}
	for _, p := range strings.Split(key, "/") {
		if p == ".." || p == "." || p == "" {
			return "", fmt.Errorf("对象路径无效")
		}
	}
	return o.prefix + key, nil
}
func (o *OSS) Put(ctx context.Context, key string, r io.Reader, size int64, ct string) error {
	k, e := o.objectKey(key)
	if e != nil {
		return e
	}
	_, e = o.client.PutObject(ctx, &oss.PutObjectRequest{Bucket: oss.Ptr(o.bucket), Key: oss.Ptr(k), Body: r, ContentLength: oss.Ptr(size), ContentType: oss.Ptr(ct), Acl: oss.ObjectACLPrivate})
	return e
}
func (o *OSS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	k, e := o.objectKey(key)
	if e != nil {
		return nil, e
	}
	r, e := o.client.GetObject(ctx, &oss.GetObjectRequest{Bucket: oss.Ptr(o.bucket), Key: oss.Ptr(k)})
	if e != nil {
		return nil, e
	}
	return r.Body, nil
}
func (o *OSS) Delete(ctx context.Context, key string) error {
	k, e := o.objectKey(key)
	if e != nil {
		return e
	}
	_, e = o.client.DeleteObject(ctx, &oss.DeleteObjectRequest{Bucket: oss.Ptr(o.bucket), Key: oss.Ptr(k)})
	return e
}
func (o *OSS) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	k, e := o.objectKey(key)
	if e != nil {
		return "", e
	}
	if ttl <= 0 || ttl > 24*time.Hour {
		return "", fmt.Errorf("签名有效期必须在 24 小时以内")
	}
	r, e := o.signer.Presign(ctx, &oss.GetObjectRequest{Bucket: oss.Ptr(o.bucket), Key: oss.Ptr(k)}, oss.PresignExpires(ttl))
	if e != nil {
		return "", e
	}
	return r.URL, nil
}
func (o *OSS) PublicURL(key string) string {
	if !IsPublicKey(key) {
		return ""
	}
	u, _ := o.SignedURL(context.Background(), key, 24*time.Hour)
	return u
}
