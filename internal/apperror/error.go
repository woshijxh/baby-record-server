package apperror

import "fmt"

const (
	CodeBadRequest   = 40001
	CodeUnauthorized = 40101
	CodeForbidden    = 40301
	CodeNotFound     = 40401
	CodeConflict     = 40901
	CodeInternal     = 50001
)

type AppError struct {
	HTTPStatus int
	Code       int
	Message    string
	Err        error
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(httpStatus, code int, message string, err error) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Err:        err,
	}
}

func BadRequest(message string, err error) *AppError {
	return New(400, CodeBadRequest, message, err)
}

func Unauthorized(message string, err error) *AppError {
	return New(401, CodeUnauthorized, message, err)
}

func Forbidden(message string, err error) *AppError {
	return New(403, CodeForbidden, message, err)
}

func NotFound(message string, err error) *AppError {
	return New(404, CodeNotFound, message, err)
}

func Conflict(message string, err error) *AppError {
	return New(409, CodeConflict, message, err)
}

func Internal(message string, err error) *AppError {
	return New(500, CodeInternal, message, err)
}
