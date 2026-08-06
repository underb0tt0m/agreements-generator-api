package domain

import (
	"errors"
	"net/http"
)

type AppErr struct {
	Msg        string
	HTTPStatus int
	Code       int
	Err        error
}

func (e *AppErr) Error() string {
	return e.Msg
}

func (e *AppErr) Wrap(msg string, err error) *AppErr {
	newErr := *e
	newErr.Err = err
	if msg != "" {
		newErr.Msg = msg
	}
	return &newErr
}

func (e *AppErr) Is(target error) bool {
	appErr, ok := errors.AsType[*AppErr](target)
	if !ok {
		return false
	}
	return e.Code == appErr.Code
}

func CheckAppErr(err error) *AppErr {
	appErr, ok := errors.AsType[*AppErr](err)
	if !ok {
		return ErrStorageBadRequest.Wrap("bad storage request", err)
	}
	return appErr
}

var (
	ErrInternal = &AppErr{
		Msg:        "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Code:       1000,
		Err:        nil,
	}
	ErrBadRequest = &AppErr{
		Msg:        "bad request",
		HTTPStatus: http.StatusBadRequest,
		Code:       1001,
		Err:        nil,
	}

	ErrStorageBadRequest = &AppErr{
		Msg:        "error during storage request",
		HTTPStatus: http.StatusInternalServerError,
		Code:       1002,
		Err:        nil,
	}
	ErrNotFound = &AppErr{
		Msg:        "source not found",
		HTTPStatus: http.StatusNotFound,
		Code:       1003,
		Err:        nil,
	}
	ErrConflict = &AppErr{
		Msg:        "conflict",
		HTTPStatus: http.StatusConflict,
		Code:       1004,
		Err:        nil,
	}
	ErrUnprocessableEntity = &AppErr{
		Msg:        "unprocessable entity",
		HTTPStatus: http.StatusUnprocessableEntity,
		Code:       1007,
		Err:        nil,
	}
	ErrUnauthorized = &AppErr{
		Msg:        "unauthorized",
		HTTPStatus: http.StatusUnauthorized,
		Code:       1008,
		Err:        nil,
	}

	ErrWrongSigningMethod = &AppErr{
		Msg:        "wrong signing method",
		HTTPStatus: http.StatusUnauthorized,
		Code:       1005,
		Err:        nil,
	}
	ErrInvalidToken = &AppErr{
		Msg:        "invalid or malformed token",
		HTTPStatus: http.StatusUnauthorized,
		Code:       1006,
		Err:        nil,
	}
)
