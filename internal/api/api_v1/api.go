package api_v1

import (
	"errors"
	"io"
	"net/http"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	"agreements-generator/internal/encoder"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/service"
	"agreements-generator/internal/token_manager"

	"github.com/go-chi/chi/v5"
)

type responseType = string

const (
	responseJSON = responseType("application/json")
	responseZIP  = responseType("application/zip")
)

type errorResponse struct {
	Details string `json:"details"`
}

type API struct {
	Log     logging.Logger
	Cfg     *config.Config
	Service *service.Generator
	Encoder encoder.Encoder
}

func (h *API) RegisterRoutes(r chi.Router, tokenMaker token_manager.TokenManager) {
	r.Group(func(r chi.Router) {
		r.Use(MWAuth(tokenMaker, h.Encoder, h.Log))
		h.BulkGenerate(r)
		h.GetJobStatus(r)
		h.GetArchiveInfo(r)
		h.GetArchive(r)
	})
}

func (h *API) BulkGenerate(r chi.Router) {
	r.Post("/bulk_generate", h.handleBulkGenerate)
}

func (h *API) handleBulkGenerate(w http.ResponseWriter, r *http.Request) {
	archiveBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Log.Error("failed to read body", logging.FieldError, err)
		writeError(w, err, h.Encoder, h.Log)
		return
	}
	defer r.Body.Close()

	h.Log.Debug("received archive", "size", len(archiveBytes))

	jobID, err := h.Service.BulkGenerate(r.Context(), archiveBytes)
	if err != nil {
		h.Log.Error("gRPC call failed",
			logging.FieldError, err.Error(),
		)
		writeError(w, err, h.Encoder, h.Log)
		return
	}
	h.Log.Info("generation started")

	responseDTO := dto.BulkGenerateResponse{JobID: jobID}

	responseBytes, err := h.Encoder.Marshal(responseDTO)
	if err != nil {
		h.Log.Error("failed to encode response body", err)
		writeError(w, domain.ErrInternal.Wrap("failed to encode response body", nil), h.Encoder, h.Log)
		return
	}

	writeResponse(w, responseBytes, responseJSON, h.Encoder, h.Log)
}

func (h *API) GetJobStatus(r chi.Router) {
	r.Get("/get_job_status", h.handleGetJobStatus)
}

func (h *API) handleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(
			w,
			domain.ErrBadRequest.Wrap(
				"query parameter ID is empty",
				nil,
			),
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
		writeError(w, domain.ErrInternal.Wrap("failed to encode response body", nil), h.Encoder, h.Log)
		return
	}

	writeResponse(w, responseBytes, responseJSON, h.Encoder, h.Log)
}

func (h *API) GetArchive(r chi.Router) {
	r.Get("/get_archive", h.handleGetArchive)
}

func (h *API) handleGetArchive(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(
			w,
			domain.ErrBadRequest.Wrap(
				"query parameter ID is empty",
				nil,
			),
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

func (h *API) GetArchiveInfo(r chi.Router) {
	r.Get("/get_archive_info", h.handleGetArchiveInfo)
}

func (h *API) handleGetArchiveInfo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(
			w,
			domain.ErrBadRequest.Wrap(
				"query parameter ID is empty",
				nil,
			),
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
		writeError(w, domain.ErrInternal.Wrap("failed to encode response body", nil), h.Encoder, h.Log)
		return
	}

	writeResponse(w, responseBytes, responseJSON, h.Encoder, h.Log)
}

func writeResponse(w http.ResponseWriter, response []byte, responseType responseType, enc encoder.Encoder, logger logging.Logger) {
	w.Header().Set("Content-Type", responseType)
	if _, err := w.Write(response); err != nil {
		logger.Error("failed to write response body", logging.FieldError, err)
		writeError(w, domain.ErrInternal.Wrap("failed to write response body", nil), enc, logger)
	}
}

func writeError(w http.ResponseWriter, err error, enc encoder.Encoder, logger logging.Logger) {
	appErr, ok := errors.AsType[*domain.AppErr](err)
	if !ok {
		w.WriteHeader(domain.ErrInternal.HTTPStatus)

		body := errorResponse{Details: domain.ErrInternal.Msg}
		responseBytes, encodeErr := enc.Marshal(body)
		if encodeErr != nil {
			logger.Error("failed to encode error response body", logging.FieldError, encodeErr)
		}

		if _, writeErr := w.Write(responseBytes); writeErr != nil {
			logger.Error("failed to write error response body", logging.FieldError, writeErr)
		}
		return
	}

	w.WriteHeader(appErr.HTTPStatus)

	body := errorResponse{Details: appErr.Msg}
	responseBytes, encodeErr := enc.Marshal(body)
	if encodeErr != nil {
		logger.Error("failed to encode error response body", logging.FieldError, encodeErr)
	}

	if _, writeErr := w.Write(responseBytes); writeErr != nil {
		logger.Error("failed to write error response body", logging.FieldError, writeErr)
	}
	return
}
