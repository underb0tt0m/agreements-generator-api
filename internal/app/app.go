package app

import (
	"fmt"
	"net/http"

	"agreements-generator/internal/api/api_v1"
	"agreements-generator/internal/config"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/service"

	"github.com/go-chi/chi/v5"
)

type App struct {
	Server        *http.Server
	CloseGRPCConn func() error
}

func New(cfg *config.Config, l logging.Logger) (*App, error) {
	gen, err := service.New(cfg, l)
	if err != nil {
		return nil, err
	}

	handler := api_v1.API{
		Log:     l,
		Cfg:     cfg,
		Service: gen,
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
