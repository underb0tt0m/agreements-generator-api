package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"agreements-generator/internal/app"
	"agreements-generator/internal/config"
	"agreements-generator/internal/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("can't load config file: %v", err)
		os.Exit(1)
	}

	logger, err := logging.Load(cfg)
	if err != nil {
		fmt.Printf("can't init logger: %v", err)
		os.Exit(1)
	}

	//	app init
	genApp, err := app.New(cfg, logger)
	if err != nil {
		fmt.Printf("init http server: %v", err)
		os.Exit(1)
	}
	defer genApp.CloseGRPCConn()

	wg := sync.WaitGroup{}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info(fmt.Sprintf("starting http server on port: %s...", cfg.Server.Port))
		genApp.Server.ListenAndServe()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-stop
		logger.Info(fmt.Sprintf("stopping http server..."))

		shutDownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second)
		defer cancel()

		if err = genApp.Server.Shutdown(shutDownCtx); err != nil {
			logger.Error(fmt.Sprintf("can't stop server gracefully: %v", err))
			return
		}
		logger.Info("server have been stopped gracefully")
	}()

	wg.Wait()
}
