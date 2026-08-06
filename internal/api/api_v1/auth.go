package api_v1

import (
	"io"
	"net/http"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	"agreements-generator/internal/encoder"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/service"

	"github.com/go-chi/chi/v5"
)

type Auth struct {
	Encoder encoder.Encoder
	Log     logging.Logger
	Service *service.Auth
}

func (a *Auth) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		a.Register(r)
		a.LogIn(r)
	})
}

func (a *Auth) Register(r chi.Router) {
	r.Post("/register", a.handleRegister)
}

func (a *Auth) handleRegister(w http.ResponseWriter, r *http.Request) {
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, domain.ErrInternal.Wrap("can't read request body", err), a.Encoder, a.Log)
		return
	}

	userData := dto.RegisterRequest{}
	if err = a.Encoder.Unmarshal(requestBody, &userData); err != nil {
		writeError(w, domain.ErrUnprocessableEntity.Wrap("can't parse request body", err), a.Encoder, a.Log)
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
		writeError(w, domain.ErrInternal.Wrap("can't marshal response body", err), a.Encoder, a.Log)
		return
	}

	writeResponse(w, response, responseJSON, a.Encoder, a.Log)
}

func (a *Auth) LogIn(r chi.Router) {
	r.Get("/login", a.handleLogIn)
}

func (a *Auth) handleLogIn(w http.ResponseWriter, r *http.Request) {
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, domain.ErrInternal.Wrap("can't read request body", err), a.Encoder, a.Log)
		return
	}

	userData := dto.LogInRequest{}
	if err = a.Encoder.Unmarshal(requestBody, &userData); err != nil {
		writeError(w, domain.ErrUnprocessableEntity.Wrap("can't parse request body", err), a.Encoder, a.Log)
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
		writeError(w, domain.ErrInternal.Wrap("can't marshal response body", err), a.Encoder, a.Log)
		return
	}

	writeResponse(w, response, responseJSON, a.Encoder, a.Log)
}
