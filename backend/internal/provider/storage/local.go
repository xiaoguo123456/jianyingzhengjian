package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Local stores objects on disk and serves them through the API's /files route.
// Private keys get an HMAC-signed, expiring URL; public prefixes are served as-is.
type Local struct {
	dir     string
	baseURL string
	secret  []byte
}

func NewLocal(dir, baseURL, secret string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Local{dir: dir, baseURL: strings.TrimRight(baseURL, "/"), secret: []byte(secret)}, nil
}

func (l *Local) path(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid key")
	}
	return filepath.Join(l.dir, clean), nil
}

func (l *Local) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *Local) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	p, err := l.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (l *Local) Delete(ctx context.Context, key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *Local) sign(key string, exp int64) string {
	m := hmac.New(sha256.New, l.secret)
	m.Write([]byte(key + "|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(m.Sum(nil))[:32]
}

func (l *Local) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	exp := time.Now().Add(ttl).Unix()
	return fmt.Sprintf("%s/files/%s?exp=%d&sig=%s", l.baseURL, key, exp, l.sign(key, exp)), nil
}

func (l *Local) PublicURL(key string) string { return l.baseURL + "/files/" + key }

// Verify checks the signature on a /files request for a private key.
func (l *Local) Verify(key string, exp int64, sig string) bool {
	if time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(l.sign(key, exp)))
}
