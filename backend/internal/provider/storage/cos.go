package storage

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// COS is the Tencent Cloud Object Storage adapter.
type COS struct {
	client    *cos.Client
	secretID  string
	secretKey string
	cdnHost   string
}

func NewCOS(bucketURL, secretID, secretKey, cdnHost string) (*COS, error) {
	u, err := url.Parse(bucketURL)
	if err != nil {
		return nil, err
	}
	c := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Timeout:   60 * time.Second,
		Transport: &cos.AuthorizationTransport{SecretID: secretID, SecretKey: secretKey},
	})
	return &COS{client: c, secretID: secretID, secretKey: secretKey, cdnHost: strings.TrimRight(cdnHost, "/")}, nil
}

func (s *COS) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opt := &cos.ObjectPutOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: contentType, ContentLength: size}}
	_, err := s.client.Object.Put(ctx, key, r, opt)
	return err
}

func (s *COS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := s.client.Object.Get(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (s *COS) Delete(ctx context.Context, key string) error {
	_, err := s.client.Object.Delete(ctx, key)
	return err
}

func (s *COS) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.client.Object.GetPresignedURL(ctx, http.MethodGet, key, s.secretID, s.secretKey, ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *COS) PublicURL(key string) string {
	if s.cdnHost != "" {
		return s.cdnHost + "/" + key
	}
	return s.client.BaseURL.BucketURL.String() + "/" + key
}
