package logger

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imrtlDrifter/template/internal/config"
	"github.com/imrtlDrifter/template/pkg/utils"
)

const (
	requestIDHeader = "X-Request-Id"
)

func Init(cfg *config.Configuration) {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(h))
}

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if utils.IsInfraPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		attrs := []any{
			"ts", start.UTC().Format(time.RFC3339Nano),
			"status", c.Writer.Status(),
			"method", c.Request.Method,
			"route", route,
			"client_ip", c.ClientIP(),
			"latency_ms", time.Since(start).Milliseconds(),
			"bytes_out", c.Writer.Size(),
		}

		if rid := requestID(c); rid != "" {
			attrs = append(attrs, "request_id", rid)
		}

		if errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String(); errMsg != "" {
			attrs = append(attrs, "gin_errors", errMsg)
		}

		if v, ok := c.Get("error_message"); ok {
			attrs = append(attrs, "error_message", fmt.Sprint(v))
		}
		if v, ok := c.Get("internal_error"); ok &&
			c.Writer.Status() >= http.StatusInternalServerError {
			attrs = append(attrs, "internal_error", fmt.Sprint(v))
		}

		slog.Log(c.Request.Context(), levelByStatus(c.Writer.Status()), "http_request", attrs...)
	}
}

func ErrorCtx(c *gin.Context, err error, msg string, extra ...any) {
	if err == nil {
		return
	}

	attrs := []any{
		"err", err.Error(),
		"method", safeMethod(c),
		"path", safePath(c),
		"client_ip", safeIP(c),
	}
	if rid := requestID(c); rid != "" {
		attrs = append(attrs, "request_id", rid)
	}
	if len(extra) > 0 {
		attrs = append(attrs, extra...)
	}

	slog.Error(msg, attrs...)
}

func InfoCtx(c *gin.Context, msg string, extra ...any) {
	attrs := []any{
		"method", safeMethod(c),
		"path", safePath(c),
		"client_ip", safeIP(c),
	}
	if rid := requestID(c); rid != "" {
		attrs = append(attrs, "request_id", rid)
	}
	if len(extra) > 0 {
		attrs = append(attrs, extra...)
	}

	slog.Info(msg, attrs...)
}

func WarnCtx(c *gin.Context, err error, msg string, extra ...any) {
	if err == nil {
		return
	}

	attrs := []any{
		"err", err.Error(),
		"method", safeMethod(c),
		"path", safePath(c),
		"client_ip", safeIP(c),
	}
	if rid := requestID(c); rid != "" {
		attrs = append(attrs, "request_id", rid)
	}
	if len(extra) > 0 {
		attrs = append(attrs, extra...)
	}

	slog.Warn(msg, attrs...)
}

func WithRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(requestIDHeader)
		if strings.TrimSpace(rid) == "" {
			rid = strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		c.Header(requestIDHeader, rid)
		c.Set("request_id", rid)
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return strings.TrimSpace(c.GetHeader(requestIDHeader))
}

func levelByStatus(code int) slog.Level {
	switch {
	case code >= http.StatusInternalServerError:
		return slog.LevelError
	case code >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func safeMethod(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return c.Request.Method
}

func safePath(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if p := c.FullPath(); p != "" {
		return p
	}
	return c.Request.URL.Path
}

func safeIP(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.ClientIP()
}
