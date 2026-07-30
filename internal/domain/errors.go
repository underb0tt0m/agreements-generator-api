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

var (
	ErrInternal = AppErr{
		Msg:        "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Code:       1000,
		Err:        nil,
	}
)
