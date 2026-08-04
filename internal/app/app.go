package app

import (
	"fmt"
	"net/http"

	"agreements-generator/internal/api/api_v1"
	"agreements-generator/internal/config"
	"agreements-generator/internal/encoder/encoder_json"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/service"
	"agreements-generator/internal/storage/storage_in_memory"

	"github.com/go-chi/chi/v5"
)

type App struct {
	Server        *http.Server
	CloseGRPCConn func() error
}

func New(cfg *config.Config, l logging.Logger) (*App, error) {
	appStorage := storage_in_memory.NewMemoryStorage(cfg)

	gen, err := service.New(cfg, l, appStorage)
	if err != nil {
		return nil, err
	}

	encoder := encoder_json.New()

	handler := api_v1.API{
		Log:     l,
		Cfg:     cfg,
		Service: gen,
		Encoder: encoder,
	}

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	APIServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	return &App{
		Server:        APIServer,
		CloseGRPCConn: gen.Close,
	}, nil
}
