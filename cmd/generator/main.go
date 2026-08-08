package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"agreements-generator/gen/go/generator"
	"agreements-generator/internal/api/api_v1"
	"agreements-generator/internal/config"
	"agreements-generator/internal/encoder/encoder_json"
	"agreements-generator/internal/gen_client"
	"agreements-generator/internal/hasher"
	loggerModule "agreements-generator/internal/logger"
	"agreements-generator/internal/service"
	"agreements-generator/internal/storage/storage_in_memory"
	"agreements-generator/internal/token_manager"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("can't load config file: %v", err)
		os.Exit(1)
	}

	logger, err := loggerModule.Load(cfg)
	if err != nil {
		fmt.Printf("can't init logger: %v", err)
		os.Exit(1)
	}

	appStorage := storage_in_memory.NewMemoryStorage(cfg)

	encoder := encoder_json.New()
	hashEr := hasher.New(cfg.Security.HashCost)
	tokenMng := token_manager.New(
		logger,
		encoder,
		cfg.JWT.TokenTTL,
		cfg.JWT.JWTSigningMethod,
		cfg.Security.SecretKey,
		cfg.JWT.Prefix,
	)

	URI := fmt.Sprintf("%s:%s", cfg.GRPCClient.Host, cfg.GRPCClient.Port)
	conn, err := grpc.NewClient(URI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("can't create GRPC Client", loggerModule.FieldError, err)
	}

	genClient := gen_client.New(generator.NewGeneratorClient(conn), conn, logger)
	defer genClient.Close()

	gen, err := service.NewGen(cfg, logger, appStorage, genClient)
	if err != nil {
		logger.Fatal("can't init service layer", loggerModule.FieldError, err)
	}
	auth := service.NewAuth(appStorage, tokenMng, hashEr)

	genHandler := api_v1.API{
		Log:     logger,
		Cfg:     cfg,
		Service: gen,
		Encoder: encoder,
	}
	authHandler := api_v1.Auth{
		Encoder: encoder,
		Log:     logger,
		Service: auth,
	}

	router := chi.NewRouter()

	genHandler.RegisterRoutes(router, tokenMng)
	authHandler.RegisterRoutes(router)

	APIServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	wg := sync.WaitGroup{}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info(fmt.Sprintf("starting http server on port: %s...", cfg.Server.Port))
		APIServer.ListenAndServe()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-stop
		logger.Info(fmt.Sprintf("stopping http server..."))

		shutDownCtx, cancel := context.WithTimeout(
			context.Background(),
			cfg.Server.ShutdownDuration)
		defer cancel()

		if err = APIServer.Shutdown(shutDownCtx); err != nil {
			logger.Error(fmt.Sprintf("can't stop server gracefully: %v", err))
			return
		}
		logger.Info("server have been stopped gracefully")
	}()

	wg.Wait()
}
