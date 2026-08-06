package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"agreements-generator/internal/app"
	"agreements-generator/internal/config"
	"agreements-generator/internal/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// есть сразу log.Fatalf()
		fmt.Printf("can't load config file: %v", err)
		os.Exit(1)
	}

	logger, err := logging.Load(cfg)
	if err != nil {
		fmt.Printf("can't init logger: %v", err)
		os.Exit(1)
	}
	/*

		Магическая функция app.New кушает конфиг и решает все сама, так нельзя делать
		подключение к (БД, kafka, redis..), подключение к внешним интеграциям, создание логгера, запуск сервера и тд
		все должно быть в main. Он для этого и нужен

		То есть у тебя должно быть так:
		1. Создал сторадж
		2. Создал грпц клиента
		3. Создал сервис - внутрь него положил сторадж и клиента (и то и то прикрыть интерфейсом нужно)
		4. Создал Хэндлер - внутрь положил сервис
		5. Запустил хттп сервер с этим хэндлером
		6. На каждом слое напилил моки и тесты
		7. Кайфанул

	*/

	//	app init
	genApp, err := app.New(cfg, logger)
	if err != nil {
		fmt.Printf("init http server: %v", err)
		os.Exit(1)
	}
	// есть общеринятый интерфейс Closer, так что метод нужно делать Close() и неважно что он там закроет
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
			cfg.Server.ShutdownDuration)
		defer cancel()

		if err = genApp.Server.Shutdown(shutDownCtx); err != nil {
			logger.Error(fmt.Sprintf("can't stop server gracefully: %v", err))
			return
		}
		logger.Info("server have been stopped gracefully")
	}()

	wg.Wait()
}
