package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/http/domainerror"
	"github.com/George1487/MalyginGeorge/internal/repository/idempotency"
)

const maxIdempotentRequestBody = 1 << 20

var errRollbackIdempotentRequest = errors.New("rollback idempotent request")

type idempotencyStore interface {
	LockAndGet(ctx context.Context, key uuid.UUID) (*idempotency.Record, error)
	Save(
		ctx context.Context,
		key uuid.UUID,
		requestHash []byte,
		responseBody []byte,
		contentType string,
		location string,
		expiresAt time.Time,
	) error
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

func IdempotencyMiddleware(
	store idempotencyStore,
	txManager txManager,
	ttl time.Duration,
) api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodPost || request.URL.Path != "/api/v1/trips" {
				next.ServeHTTP(w, request)
				return
			}

			keyText := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
			if keyText == "" {
				next.ServeHTTP(w, request)
				return
			}

			key, err := uuid.Parse(keyText)
			if err != nil {
				domainerror.Write(w, trip.ErrInvalidRequest)
				return
			}

			bodyReader := http.MaxBytesReader(w, request.Body, maxIdempotentRequestBody)
			body, err := io.ReadAll(bodyReader)
			_ = bodyReader.Close()
			if err != nil {
				domainerror.Write(w, trip.ErrInvalidRequest)
				return
			}

			requestHash := sha256.Sum256(body)
			var response *bufferedResponse
			var replay *idempotency.Record
			conflict := false

			err = txManager.Do(request.Context(), func(txCtx context.Context) error {
				record, getErr := store.LockAndGet(txCtx, key)
				if getErr != nil {
					return getErr
				}
				if record != nil {
					if !bytes.Equal(record.RequestHash, requestHash[:]) {
						conflict = true
						return nil
					}

					replay = record
					return nil
				}

				response = newBufferedResponse()
				requestCopy := request.Clone(txCtx)
				requestCopy.Body = io.NopCloser(bytes.NewReader(body))
				next.ServeHTTP(response, requestCopy)

				if response.StatusCode() != http.StatusCreated {
					return errRollbackIdempotentRequest
				}

				if saveErr := store.Save(
					txCtx,
					key,
					requestHash[:],
					response.body.Bytes(),
					response.header.Get("Content-Type"),
					response.header.Get("Location"),
					time.Now().UTC().Add(ttl),
				); saveErr != nil {
					return saveErr
				}

				return nil
			})

			switch {
			case errors.Is(err, errRollbackIdempotentRequest):
				response.WriteTo(w, response.StatusCode())
			case err != nil:
				domainerror.Write(w, err)
			case conflict:
				domainerror.Write(w, trip.ErrIdempotencyConflict)
			case replay != nil:
				writeStoredResponse(w, replay)
			default:
				response.WriteTo(w, response.StatusCode())
			}
		})
	}
}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{header: make(http.Header)}
}

func (response *bufferedResponse) Header() http.Header {
	return response.header
}

func (response *bufferedResponse) WriteHeader(status int) {
	if response.status == 0 {
		response.status = status
	}
}

func (response *bufferedResponse) Write(data []byte) (int, error) {
	if response.status == 0 {
		response.status = http.StatusOK
	}

	return response.body.Write(data)
}

func (response *bufferedResponse) StatusCode() int {
	if response.status == 0 {
		return http.StatusOK
	}

	return response.status
}

func (response *bufferedResponse) WriteTo(w http.ResponseWriter, status int) {
	copyHeaders(w.Header(), response.header)
	w.WriteHeader(status)
	_, _ = w.Write(response.body.Bytes())
}

func writeStoredResponse(w http.ResponseWriter, record *idempotency.Record) {
	if record.ContentType != "" {
		w.Header().Set("Content-Type", record.ContentType)
	}
	if record.Location != "" {
		w.Header().Set("Location", record.Location)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(record.ResponseBody)
}

func copyHeaders(destination, source http.Header) {
	for name, values := range source {
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}
