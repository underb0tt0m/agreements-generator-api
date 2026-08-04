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

func (h *API) RegisterRoutes(r chi.Router) {
	h.BulkGenerate(r)
	h.GetJobStatus(r)
	h.GetArchiveInfo(r)
	h.GetArchive(r)
}

func (h *API) BulkGenerate(r chi.Router) {
	r.Post("/bulk_generate", h.handleBulkGenerate)
}

func (h *API) handleBulkGenerate(w http.ResponseWriter, r *http.Request) {
	archiveBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.Log.Error("failed to read body", "error", err)
		h.writeError(w, err)
		return
	}
	defer r.Body.Close()

	h.Log.Debug("received archive", "size", len(archiveBytes))

	jobID, err := h.Service.BulkGenerate(r.Context(), archiveBytes)
	if err != nil {
		h.Log.Error("gRPC call failed",
			"error", err.Error(),
		)
		h.writeError(w, err)
		return
	}
	h.Log.Info("generation started")

	responseDTO := dto.BulkGenerateResponse{JobID: jobID}

	responseBytes, err := h.Encoder.Marshal(responseDTO)
	if err != nil {
		h.Log.Error("failed to encode response body", err)
		h.writeError(w, domain.ErrInternal.Wrap("failed to encode response body", nil))
		return
	}

	h.writeResponse(w, responseBytes, responseJSON)
}

func (h *API) GetJobStatus(r chi.Router) {
	r.Get("/get_job_status", h.handleGetJobStatus)
}

func (h *API) handleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		h.writeError(w, domain.ErrBadRequest.Wrap(
			"query parameter ID is empty",
			nil,
		))
		return
	}

	jobStatus, err := h.Service.CheckJobStatus(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}

	responseDTO := dto.GetJobStatusResponse{
		JobID:  id,
		Status: string(jobStatus),
	}

	responseBytes, err := h.Encoder.Marshal(responseDTO)
	if err != nil {
		h.Log.Error("failed to encode response body", err)
		h.writeError(w, domain.ErrInternal.Wrap("failed to encode response body", nil))
		return
	}

	h.writeResponse(w, responseBytes, responseJSON)
}

func (h *API) GetArchive(r chi.Router) {
	r.Get("/get_archive", h.handleGetArchive)
}

func (h *API) handleGetArchive(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		h.writeError(w, domain.ErrBadRequest.Wrap(
			"query parameter ID is empty",
			nil,
		))
		return
	}

	archive, err := h.Service.GetArchive(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeResponse(w, archive, responseZIP)
}

func (h *API) GetArchiveInfo(r chi.Router) {
	r.Get("/get_archive_info", h.handleGetArchiveInfo)
}

func (h *API) handleGetArchiveInfo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		h.writeError(w, domain.ErrBadRequest.Wrap(
			"query parameter ID is empty",
			nil,
		))
		return
	}

	genErrs, genCnt, err := h.Service.GetArchiveInfo(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}

	responseDTO := dto.NewGetArchiveInfoResponse(genErrs, genCnt)

	responseBytes, err := h.Encoder.Marshal(responseDTO)
	if err != nil {
		h.Log.Error("failed to encode response body", err)
		h.writeError(w, domain.ErrInternal.Wrap("failed to encode response body", nil))
		return
	}

	h.writeResponse(w, responseBytes, responseJSON)
}

func (h *API) writeResponse(w http.ResponseWriter, response []byte, responseType responseType) {
	w.Header().Set("Content-Type", responseType)
	if _, err := w.Write(response); err != nil {
		h.Log.Error("failed to write response body", "error", err)
		h.writeError(w, domain.ErrInternal.Wrap("failed to write response body", nil))
	}
}

func (h *API) writeError(w http.ResponseWriter, err error) {
	appErr, ok := errors.AsType[*domain.AppErr](err)
	if !ok {
		w.WriteHeader(domain.ErrInternal.HTTPStatus)

		body := errorResponse{Details: domain.ErrInternal.Msg}
		responseBytes, encodeErr := h.Encoder.Marshal(body)
		if encodeErr != nil {
			h.Log.Error("failed to encode error response body", "error", encodeErr)
		}

		if _, writeErr := w.Write(responseBytes); writeErr != nil {
			h.Log.Error("failed to write error response body", "error", writeErr)
		}
		return
	}

	w.WriteHeader(appErr.HTTPStatus)

	body := errorResponse{Details: appErr.Msg}
	responseBytes, encodeErr := h.Encoder.Marshal(body)
	if encodeErr != nil {
		h.Log.Error("failed to encode error response body", "error", encodeErr)
	}

	if _, writeErr := w.Write(responseBytes); writeErr != nil {
		h.Log.Error("failed to write error response body", "error", writeErr)
	}
	return
}
