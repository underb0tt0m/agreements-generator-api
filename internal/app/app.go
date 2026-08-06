package app

import (
	"fmt"
	"net/http"

	"agreements-generator/internal/api/api_v1"
	"agreements-generator/internal/config"
	"agreements-generator/internal/encoder/encoder_json"
	"agreements-generator/internal/hasher"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/service"
	"agreements-generator/internal/storage/storage_in_memory"
	"agreements-generator/internal/token_manager"

	"github.com/go-chi/chi/v5"
)

type App struct {
	Server        *http.Server
	CloseGRPCConn func() error
}

func New(cfg *config.Config, l logging.Logger) (*App, error) {
	appStorage := storage_in_memory.NewMemoryStorage(cfg)
	encoder := encoder_json.New()
	hashEr := hasher.New(cfg.Security.HashCost)
	tokenMng := token_manager.New(
		l,
		encoder,
		cfg.JWT.TokenTTL,
		cfg.JWT.JWTSigningMethod,
		cfg.Security.SecretKey,
		cfg.JWT.Prefix,
	)

	gen, err := service.NewGen(cfg, l, appStorage)
	if err != nil {
		return nil, err
	}
	auth := service.NewAuth(appStorage, tokenMng, hashEr)

	genHandler := api_v1.API{
		Log:     l,
		Cfg:     cfg,
		Service: gen,
		Encoder: encoder,
	}
	authHandler := api_v1.Auth{
		Encoder: encoder,
		Log:     l,
		Service: auth,
	}

	router := chi.NewRouter()

	router.Use()

	genHandler.RegisterRoutes(router, tokenMng)
	authHandler.RegisterRoutes(router)

	APIServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	return &App{
		Server:        APIServer,
		CloseGRPCConn: gen.Close,
	}, nil
}
