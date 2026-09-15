// Package storage abstracts object storage (docs/BACKEND_ARCHITECTURE.md §7).
package storage

import (
	"context"
	"io"
	"strings"
	"time"
)

type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	// SignedURL returns a time-limited URL for a private object.
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	// PublicURL returns the CDN/public URL for keys under public prefixes (assets/, shares/).
	PublicURL(key string) string
}

// IsPublicKey reports whether the key lives under a public prefix.
func IsPublicKey(key string) bool {
	return strings.HasPrefix(key, "assets/") || strings.HasPrefix(key, "shares/")
}

// URLFor picks public or signed URL by prefix.
func URLFor(ctx context.Context, s ObjectStore, key string, ttl time.Duration) string {
	if key == "" {
		return ""
	}
	if IsPublicKey(key) {
		return s.PublicURL(key)
	}
	u, err := s.SignedURL(ctx, key, ttl)
	if err != nil {
		return ""
	}
	return u
}

// ContentTypeFor maps an output format to a MIME type.
func ContentTypeFor(format string) string {
	switch strings.ToLower(format) {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
