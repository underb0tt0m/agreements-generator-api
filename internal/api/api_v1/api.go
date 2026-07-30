package api_v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/service"

	"github.com/go-chi/chi/v5"
)

type API struct {
	Log     logging.Logger
	Cfg     *config.Config
	Service *service.Generator
}

func (h *API) RegisterRoutes(r chi.Router) {
	h.BulkGenerate(r)
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

	outputArchive, errs, genCount, fatalErr := h.Service.BulkGenerate(r.Context(), archiveBytes)
	if fatalErr != nil {
		h.Log.Error("gRPC call failed",
			"error", fatalErr.Error(),
		)
		h.writeError(w, fatalErr)
		return
	}
	h.Log.Info("generation completed", "errors", len(errs), "count", genCount)

	responseDTO := dto.NewBulkGenerateResponse(outputArchive, errs, genCount)

	h.Log.Debug("response",
		"archive_len", len(responseDTO.Archive),
		"errors", len(responseDTO.ErrorsList),
		"gen_count", responseDTO.GenCount,
	)

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(responseDTO); err != nil {
		h.Log.Error("failed to encode response", "error", err)
		h.writeError(w, err)
		return
	}

}

func (h *API) writeError(w http.ResponseWriter, err error) {
	appErr, ok := errors.AsType[*domain.AppErr](err)
	if !ok {
		w.WriteHeader(domain.ErrInternal.HTTPStatus)
		body := fmt.Sprintf("details: %s", domain.ErrInternal.Msg)
		if err = json.NewEncoder(w).Encode(body); err != nil {
			h.Log.Error("failed to encode error response body", "error", err)
		}
		return
	}
	w.WriteHeader(appErr.HTTPStatus)
	body := fmt.Sprintf("details: %s", appErr.Msg)
	if err = json.NewEncoder(w).Encode(body); err != nil {
		h.Log.Error("failed to encode error response body", "error", err)
	}
}
