package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	httprouter "github.com/bestows-Z/dev-hub/backend/internal/http/router"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(
			"load config failed",
			zap.Error(err),
		)
	}
	postgresClient, err := pg.New(
		pg.Config{
			DSN:          cfg.Postgres.DSN(),
			MaxOpenConns: cfg.Postgres.MaxOpenConns,
			MaxIdleConns: cfg.Postgres.MaxIdleConns,
		},
		logger,
	)
	if err != nil {
		logger.Fatal(
			"initialize postgres failed",
			zap.Error(err),
		)
	}
	userRepository := user.NewRepository(
		postgresClient.DB,
	)

	userService := user.NewService(
		userRepository,
	)
	userHandler := user.NewHandler(
		userService,
		logger,
	)
	router := httprouter.New(postgresClient.SQLDB, userHandler)
	address := fmt.Sprintf(
		"%s:%d",
		cfg.HTTP.Host,
		cfg.HTTP.Port,
	)
	server := &http.Server{
		Addr:    address,
		Handler: router,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info(
			"devhub api started",
			zap.String(
				"address",
				address,
			),
		)

		err := server.ListenAndServe()

		if err != nil &&
			!errors.Is(
				err,
				http.ErrServerClosed,
			) {

			logger.Error(
				"http server failed",
				zap.Error(err),
			)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info(
		"shutting down devhub api",
	)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(
			"http server shutdown failed",
			zap.Error(err),
		)
	}
	if err := postgresClient.Close(); err != nil {
		logger.Error(
			"close postgres failed",
			zap.Error(err),
		)
	}

	logger.Info(
		"devhub api stopped",
	)
}
