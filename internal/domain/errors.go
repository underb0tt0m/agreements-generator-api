package domain

import (
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

func (e *AppErr) Unwrap() error {
	return e.Err
}

var (
	ErrInternal = &AppErr{
		Msg:        "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Code:       1000,
	}
	ErrBadRequest = &AppErr{
		Msg:        "bad request",
		HTTPStatus: http.StatusBadRequest,
		Code:       1001,
	}
	ErrNotFound = &AppErr{
		Msg:        "source not found",
		HTTPStatus: http.StatusNotFound,
		Code:       1003,
	}
	ErrConflict = &AppErr{
		Msg:        "conflict",
		HTTPStatus: http.StatusConflict,
		Code:       1004,
	}
	ErrUnprocessableEntity = &AppErr{
		Msg:        "unprocessable entity",
		HTTPStatus: http.StatusUnprocessableEntity,
		Code:       1007,
	}
	ErrUnauthorized = &AppErr{
		Msg:        "unauthorized",
		HTTPStatus: http.StatusUnauthorized,
		Code:       1008,
	}

	ErrStorageBadRequest = &AppErr{
		Msg:        "error during storage request",
		HTTPStatus: http.StatusInternalServerError,
		Code:       1002,
	}
	ErrWrongSigningMethod = &AppErr{
		Msg:        "wrong signing method",
		HTTPStatus: http.StatusUnauthorized,
		Code:       1005,
	}
	ErrInvalidToken = &AppErr{
		Msg:        "invalid or malformed token",
		HTTPStatus: http.StatusUnauthorized,
		Code:       1006,
	}
	ErrGenClient = &AppErr{
		Msg:        "error during generation",
		HTTPStatus: http.StatusInternalServerError,
		Code:       1009,
	}
	ErrJobNotFinished = &AppErr{
		Msg:        "job not finished",
		HTTPStatus: http.StatusBadRequest,
		Code:       1010,
	}
	ErrHashComparing = &AppErr{
		Msg:        "wrong login or password",
		HTTPStatus: http.StatusUnauthorized,
		Code:       1011,
	}
)
