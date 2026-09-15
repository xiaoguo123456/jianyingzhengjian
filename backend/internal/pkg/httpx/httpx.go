// Package httpx holds the response envelope helpers for gin handlers.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"yingji/backend/internal/pkg/apperr"
)

const (
	CtxRequestID = "request_id"
	CtxUserID    = "user_id"
	CtxAdminID   = "admin_id"
	CtxAdminRole = "admin_role"
)

type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

func RequestID(c *gin.Context) string { return c.GetString(CtxRequestID) }
func UserID(c *gin.Context) string    { return c.GetString(CtxUserID) }
func AdminID(c *gin.Context) string   { return c.GetString(CtxAdminID) }

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: "OK", Data: data, RequestID: RequestID(c)})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Code: "OK", Data: data, RequestID: RequestID(c)})
}

func Accepted(c *gin.Context, data any) {
	c.JSON(http.StatusAccepted, Envelope{Code: "OK", Data: data, RequestID: RequestID(c)})
}

// Fail writes an error envelope; unknown errors become INTERNAL and are logged.
func Fail(c *gin.Context, err error) {
	e := apperr.From(err)
	if e.Status >= 500 {
		slog.Error("request failed", "request_id", RequestID(c), "path", c.FullPath(), "err", err)
	}
	c.AbortWithStatusJSON(e.Status, Envelope{Code: e.Code, Message: e.Message, Data: e.Data, RequestID: RequestID(c)})
}

// Bind binds JSON and converts validation failures to BAD_REQUEST.
func Bind(c *gin.Context, v any) error {
	if err := c.ShouldBindJSON(v); err != nil {
		return apperr.BadRequest("参数错误").WithData(map[string]any{"fields": err.Error()})
	}
	return nil
}

func IsNotFound(err error) bool { return errors.Is(err, errNotFound) }

var errNotFound = errors.New("not found")
