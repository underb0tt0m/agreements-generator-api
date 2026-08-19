package api_v1

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	"agreements-generator/internal/encoder"
	"agreements-generator/internal/logger"
	"agreements-generator/internal/service"
	"agreements-generator/internal/token_manager"

	"github.com/go-chi/chi/v5"
)

type responseType = string

const (
	responseJSON = responseType("application/json")
	responseZIP  = responseType("application/zip")
)

type API struct {
	Log     logger.Logger
	Service service.Generator
	Encoder encoder.Encoder
}

func (h *API) RegisterRoutes(r chi.Router, tokenMaker token_manager.TokenManager) {
	r.Group(func(r chi.Router) {
		r.Use(MWAuth(tokenMaker, h.Encoder, h.Log))
		r.Post("/bulk_generate", h.handleBulkGenerate)
		r.Get("/get_job_status", h.handleGetJobStatus)
		r.Get("/get_archive_info", h.handleGetArchiveInfo)
		r.Get("/get_archive", h.handleGetArchive)
	})
}

func (h *API) handleBulkGenerate(w http.ResponseWriter, r *http.Request) {
	archiveBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Log.Error("failed to read body", logger.FieldError, err)
		writeError(w, err, h.Encoder, h.Log)
		return
	}
	defer r.Body.Close()

	h.Log.Debug("received archive", "size", len(archiveBytes))

	jobID, err := h.Service.BulkGenerate(r.Context(), archiveBytes)
	if err != nil {
		h.Log.Error("failed request",
			logger.FieldError, err.Error(),
		)
		writeError(w, err, h.Encoder, h.Log)
		return
	}
	h.Log.Info("generation started")

	responseDTO := dto.BulkGenerateResponse{JobID: jobID}

	responseBytes, err := h.Encoder.Marshal(responseDTO)
	if err != nil {
		h.Log.Error("failed to encode response body", err)
		writeError(
			w,
			fmt.Errorf("failed to encode response body: %v, %w", err, domain.ErrInternal),
			h.Encoder,
			h.Log,
		)
		return
	}

	writeResponse(w, responseBytes, responseJSON, h.Encoder, h.Log)
}

func (h *API) handleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(
			w,
			fmt.Errorf("query parameter ID is empty: %w", domain.ErrBadRequest),
			h.Encoder,
			h.Log,
		)
		return
	}

	jobStatus, err := h.Service.CheckJobStatus(r.Context(), id)
	if err != nil {
		writeError(w, err, h.Encoder, h.Log)
		return
	}

	responseDTO := dto.GetJobStatusResponse{
		JobID:  id,
		Status: string(jobStatus),
	}

	responseBytes, err := h.Encoder.Marshal(responseDTO)
	if err != nil {
		h.Log.Error("failed to encode response body", err)
		writeError(
			w,
			fmt.Errorf("failed to encode response body: %v, %w", err, domain.ErrInternal),
			h.Encoder,
			h.Log,
		)
		return
	}

	writeResponse(w, responseBytes, responseJSON, h.Encoder, h.Log)
}

func (h *API) handleGetArchive(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(
			w,
			fmt.Errorf("query parameter ID is empty: %w", domain.ErrBadRequest),
			h.Encoder,
			h.Log,
		)
		return
	}

	archive, err := h.Service.GetArchive(r.Context(), id)
	if err != nil {
		writeError(w, err, h.Encoder, h.Log)
		return
	}

	writeResponse(w, archive, responseZIP, h.Encoder, h.Log)
}

func (h *API) handleGetArchiveInfo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(
			w,
			fmt.Errorf("query parameter ID is empty: %w", domain.ErrBadRequest),
			h.Encoder,
			h.Log,
		)
		return
	}

	genErrs, genCnt, err := h.Service.GetArchiveInfo(r.Context(), id)
	if err != nil {
		writeError(w, err, h.Encoder, h.Log)
		return
	}

	responseDTO := dto.NewGetArchiveInfoResponse(genErrs, genCnt)

	responseBytes, err := h.Encoder.Marshal(responseDTO)
	if err != nil {
		h.Log.Error("failed to encode response body", err)
		writeError(
			w,
			fmt.Errorf("failed to encode response body: %v, %w", err, domain.ErrInternal),
			h.Encoder,
			h.Log,
		)
		return
	}

	writeResponse(w, responseBytes, responseJSON, h.Encoder, h.Log)
}

func writeResponse(w http.ResponseWriter, response []byte, responseType responseType, enc encoder.Encoder, l logger.Logger) {
	w.Header().Set("Content-Type", responseType)
	if _, err := w.Write(response); err != nil {
		l.Error("failed to write response body", logger.FieldError, err)
		writeError(
			w,
			fmt.Errorf("failed to write response body: %v, %w", err, domain.ErrInternal),
			enc,
			l,
		)
	}
}

func writeError(w http.ResponseWriter, err error, enc encoder.Encoder, l logger.Logger) {
	appErr, ok := errors.AsType[*domain.AppErr](err)
	if !ok {
		l.Error("unexpected error", logger.FieldError, err)

		w.WriteHeader(domain.ErrInternal.HTTPStatus)

		body := dto.ErrorResponse{Details: domain.ErrInternal.Msg}
		responseBytes, encodeErr := enc.Marshal(body)
		if encodeErr != nil {
			l.Error("failed to encode error response body", logger.FieldError, encodeErr)
		}

		if _, writeErr := w.Write(responseBytes); writeErr != nil {
			l.Error("failed to write error response body", logger.FieldError, writeErr)
		}
		return
	}

	l.Debug("error during request", logger.FieldError, err)

	w.WriteHeader(appErr.HTTPStatus)

	body := dto.ErrorResponse{Details: appErr.Msg}
	responseBytes, encodeErr := enc.Marshal(body)
	if encodeErr != nil {
		l.Error("failed to encode error response body", logger.FieldError, encodeErr)
	}

	if _, writeErr := w.Write(responseBytes); writeErr != nil {
		l.Error("failed to write error response body", logger.FieldError, writeErr)
	}
	return
}
