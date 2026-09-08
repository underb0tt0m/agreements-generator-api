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
	"agreements-generator/internal/cache"
	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/encoder/encoder_json"
	"agreements-generator/internal/gen_client"
	"agreements-generator/internal/hasher"
	loggerModule "agreements-generator/internal/logger"
	"agreements-generator/internal/publisher"
	"agreements-generator/internal/service"
	"agreements-generator/internal/storage"
	"agreements-generator/internal/token_manager"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	generatorStorage, userStorage, err := storage.New(context.Background(), cfg, logger, encoder)
	if err != nil {
		logger.Fatal("can't create storage", loggerModule.FieldError, err)
	}

	URI := fmt.Sprintf("%s:%s", cfg.GRPCClient.Host, cfg.GRPCClient.Port)
	conn, err := grpc.NewClient(URI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("can't create GRPC Client", loggerModule.FieldError, err)
	}

	grpcClient := gen_client.New(generator.NewGeneratorClient(conn), conn, logger)
	domain.CloseObj(grpcClient, logger)

	cacher := cache.New(cfg.Redis.Host, cfg.Redis.Port, cfg.Security.RedisPassword, cfg.Redis.Db, cfg.Redis.JobStatusTTL)
	domain.CloseObj(cacher, logger)

	publer, err := publisher.New(
		cfg.RabbitMQ.Host,
		cfg.RabbitMQ.Port,
		cfg.RabbitMQ.Username,
		cfg.Security.RabbitMQPassword,
		cfg.RabbitMQ.Vhost,
		cfg.RabbitMQ.Queue,
		encoder,
	)
	if err != nil {
		logger.Fatal("can't create publisher", loggerModule.FieldError, err)
	}
	domain.CloseObj(publer, logger)

	gen, err := service.NewGen(
		cfg.ExecMod,
		logger,
		generatorStorage,
		grpcClient,
		publer,
		cacher,
		cfg.GRPCClient.JobMaxDuration,
	)
	if err != nil {
		logger.Fatal("can't init service layer", loggerModule.FieldError, err)
	}
	auth := service.NewAuth(userStorage, tokenMng, hashEr, cfg.Security.MinPasswordLen)

	genHandler := api_v1.API{
		Log:     logger,
		Service: gen,
		Encoder: encoder,
	}
	authHandler := api_v1.Auth{
		Encoder: encoder,
		Log:     logger,
		Service: auth,
	}

	router := chi.NewRouter()

	router.Use(api_v1.MWMetrics)
	router.Handle("/metrics", promhttp.Handler())

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
		if err = APIServer.ListenAndServe(); err != nil {
			logger.Error("can't start http server", loggerModule.FieldError, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-stop
		logger.Info("stopping http server...")

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
