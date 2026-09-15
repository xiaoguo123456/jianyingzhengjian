// Package apperr defines the API error type (docs/API.md §1).
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Code      string
	Status    int
	Message   string
	Data      any
	Transient bool
	cause     error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

func New(code string, status int, msg string) *Error {
	return &Error{Code: code, Status: status, Message: msg}
}

func (e *Error) WithData(d any) *Error       { c := *e; c.Data = d; return &c }
func (e *Error) WithCause(err error) *Error  { c := *e; c.cause = err; return &c }
func (e *Error) WithMessage(m string) *Error { c := *e; c.Message = m; return &c }

func BadRequest(msg string) *Error { return New("BAD_REQUEST", http.StatusBadRequest, msg) }
func Unauthorized() *Error {
	return New("UNAUTHORIZED", http.StatusUnauthorized, "登录已过期，请重试")
}
func Forbidden() *Error           { return New("FORBIDDEN", http.StatusForbidden, "没有权限") }
func NotFound(what string) *Error { return New("NOT_FOUND", http.StatusNotFound, what+"不存在") }
func Conflict(msg string) *Error  { return New("CONFLICT", http.StatusConflict, msg) }
func RateLimited() *Error {
	return New("RATE_LIMITED", http.StatusTooManyRequests, "操作太频繁，请稍后再试")
}
func PayloadTooLarge() *Error {
	return New("PAYLOAD_TOO_LARGE", http.StatusRequestEntityTooLarge, "文件过大")
}

func NoCredits(credits any) *Error {
	return New("NO_CREDITS", http.StatusPaymentRequired, "生成次数不足").WithData(map[string]any{"credits": credits})
}

func PhotoRejected(reasons []string) *Error {
	return New("PHOTO_REJECTED", http.StatusUnprocessableEntity, "这张照片可能影响生成效果，请换一张清晰正脸照片。").
		WithData(map[string]any{"reasons": reasons})
}

func AdSessionInvalid(reason string) *Error {
	return New("AD_SESSION_INVALID", http.StatusUnprocessableEntity, "未完整观看视频，未获得次数").
		WithData(map[string]any{"reason": reason})
}

func GenerationUnavailable() *Error {
	return New("GENERATION_UNAVAILABLE", http.StatusServiceUnavailable, "生成服务暂时不可用，请稍后再试。")
}

func Internal(err error) *Error {
	return New("INTERNAL", http.StatusInternalServerError, "出了点问题，请稍后再试").WithCause(err)
}

// Transient marks a provider/network failure worth retrying.
func Transient(code string, err error) *Error {
	e := New(code, http.StatusBadGateway, "upstream error").WithCause(err)
	e.Transient = true
	return e
}

// From converts any error to *Error.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Internal(err)
}

func Is(err error, code string) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}

func IsTransient(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Transient
}
