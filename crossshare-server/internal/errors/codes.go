package errors

import "net/http"

type AppError struct {
	HTTPStatus int
	Code       int
	Message    string
}

func (e *AppError) Error() string {
	return e.Message
}

var (
	ErrInvalidPayload    = &AppError{http.StatusBadRequest, 1001, "invalid request payload"}
	ErrPayloadTooLarge   = &AppError{http.StatusRequestEntityTooLarge, 1002, "payload too large"}
	ErrInvalidTTL        = &AppError{http.StatusBadRequest, 1003, "invalid ttl"}
	ErrUnsupportedType   = &AppError{http.StatusUnsupportedMediaType, 1004, "unsupported content-type"}
	ErrInvalidKey        = &AppError{http.StatusBadRequest, 1101, "invalid key format"}
	ErrNotFound          = &AppError{http.StatusNotFound, 1404, "share not found"}
	ErrStorageInternal   = &AppError{http.StatusInternalServerError, 1500, "internal storage error"}
	ErrAuthInvalid       = &AppError{http.StatusUnauthorized, 1601, "auth token invalid"}
	ErrRateLimitExceeded = &AppError{http.StatusTooManyRequests, 1701, "rate limit exceeded"}
)
