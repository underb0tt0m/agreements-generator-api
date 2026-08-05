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

/*
	эта функиця не нужна
	глобально ошибки двух видов: ожидаемые и нет
	проверять их нужно только в 1 месте - handler и там делать вывод, что отдать клиенту
	1. Если ошибка твоя - то код и текст
	2. Во всех остальных случаях 500/InternalServerError/Произошла непредвиденная ошибка
*/

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
	}
	ErrBadRequest = &AppErr{
		Msg:        "bad request",
		HTTPStatus: http.StatusBadRequest,
		Code:       1001,
	}

	ErrStorageBadRequest = &AppErr{
		Msg:        "error during storage request",
		HTTPStatus: http.StatusInternalServerError,
		Code:       1002,
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
)
