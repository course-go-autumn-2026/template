package http_res

import (
	"errors"
	"net/http"
	"strings"

	api "github.com/imrtlDrifter/template/internal/generated"

	"github.com/gin-gonic/gin"
)

const problemTypeBaseURL = "https://tripgo.example/problems/"

type HTTPError = api.Problem

type appError struct {
	err        error
	httpStatus int
	code       string
	title      string
	detail     string
}

func (e *appError) Error() string   { return e.err.Error() }
func (e *appError) Unwrap() error   { return e.err }
func (e *appError) HTTPStatus() int { return e.httpStatus }
func (e *appError) Code() string    { return e.code }
func (e *appError) Title() string   { return e.title }
func (e *appError) Detail() string  { return e.detail }

func NewHTTPError(err error, status int, code, title, detail string) error {
	return &appError{
		err:        err,
		httpStatus: status,
		code:       code,
		title:      title,
		detail:     detail,
	}
}

func NewError(ctx *gin.Context, status int, err error) {
	code := "invalid_request"
	title := "Invalid request"
	detail := err.Error()

	var mapped interface {
		error
		Code() string
		Title() string
		Detail() string
	}
	if errors.As(err, &mapped) {
		code = mapped.Code()
		title = mapped.Title()
		detail = mapped.Detail()
	}
	if status >= http.StatusInternalServerError {
		code = "internal_error"
		title = "Internal Server Error"
		detail = "Internal server error"
	}

	instance := ctx.Request.URL.Path
	problemType := problemTypeBaseURL + strings.ReplaceAll(code, "_", "-")

	ctx.Header("Content-Type", "application/problem+json")
	ctx.JSON(status, HTTPError{
		Type:     problemType,
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	})
	ctx.Abort()
}

func HandleDomainError(ctx *gin.Context, err error) {
	if err == nil {
		return
	}

	var mapped interface {
		error
		HTTPStatus() int
		Code() string
		Title() string
		Detail() string
	}

	if errors.As(err, &mapped) {
		ctx.Set("error_message", mapped.Detail())
		NewError(ctx, mapped.HTTPStatus(), err)
		return
	}

	_ = ctx.Error(err).SetType(gin.ErrorTypePrivate)
	ctx.Set("error_message", "Internal server error")
	ctx.Set("internal_error", err.Error())

	NewError(ctx, http.StatusInternalServerError, err)
}
