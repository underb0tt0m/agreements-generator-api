package api_v1

import (
	"fmt"
	"io"
	"net/http"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	"agreements-generator/internal/encoder"
	"agreements-generator/internal/logger"
	"agreements-generator/internal/service"

	"github.com/go-chi/chi/v5"
)

type Auth struct {
	Encoder encoder.Encoder
	Log     logger.Logger
	Service *service.Auth
}

func (a *Auth) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", a.handleRegister)
		r.Get("/login", a.handleLogIn)
	})
}

func (a *Auth) handleRegister(w http.ResponseWriter, r *http.Request) {
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(
			w,
			fmt.Errorf("can't read request body: %v, %w", err, domain.ErrInternal),
			a.Encoder,
			a.Log,
		)
		return
	}

	userData := dto.RegisterRequest{}
	if err = a.Encoder.Unmarshal(requestBody, &userData); err != nil {
		writeError(
			w,
			fmt.Errorf("can't parse request body: %v, %w", err, domain.ErrUnprocessableEntity),
			a.Encoder,
			a.Log,
		)
		return
	}

	token, err := a.Service.Register(r.Context(), userData)
	if err != nil {
		writeError(w, err, a.Encoder, a.Log)
		return
	}

	tokenDTO := dto.AuthResponse{Token: token}

	response, err := a.Encoder.Marshal(tokenDTO)
	if err != nil {
		writeError(
			w,
			fmt.Errorf("can't marshal response body: %v, %w", err, domain.ErrInternal),
			a.Encoder,
			a.Log,
		)
		return
	}

	writeResponse(w, response, responseJSON, a.Encoder, a.Log)
}

func (a *Auth) handleLogIn(w http.ResponseWriter, r *http.Request) {
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(
			w,
			fmt.Errorf("can't read request body: %v, %w", err, domain.ErrInternal),
			a.Encoder,
			a.Log,
		)
		return
	}

	userData := dto.LogInRequest{}
	if err = a.Encoder.Unmarshal(requestBody, &userData); err != nil {
		writeError(
			w,
			fmt.Errorf("can't parse request body: %v, %w", err, domain.ErrUnprocessableEntity),
			a.Encoder,
			a.Log,
		)
		return
	}

	token, err := a.Service.LogIn(r.Context(), userData)
	if err != nil {
		writeError(w, err, a.Encoder, a.Log)
		return
	}

	tokenDTO := dto.AuthResponse{Token: token}

	response, err := a.Encoder.Marshal(tokenDTO)
	if err != nil {
		writeError(
			w,
			fmt.Errorf("can't marshal response body: %v, %w", err, domain.ErrInternal),
			a.Encoder,
			a.Log,
		)
		return
	}

	writeResponse(w, response, responseJSON, a.Encoder, a.Log)
}
