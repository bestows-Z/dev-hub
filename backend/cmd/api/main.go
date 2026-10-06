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

	"github.com/bestows-Z/dev-hub/backend/internal/admin"
	"github.com/bestows-Z/dev-hub/backend/internal/analytics"
	"github.com/bestows-Z/dev-hub/backend/internal/assistant"
	"github.com/bestows-Z/dev-hub/backend/internal/auth"
	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/bestows-Z/dev-hub/backend/internal/content"
	"github.com/bestows-Z/dev-hub/backend/internal/engagement"
	"github.com/bestows-Z/dev-hub/backend/internal/gallery"
	httprouter "github.com/bestows-Z/dev-hub/backend/internal/http/router"
	"github.com/bestows-Z/dev-hub/backend/internal/media"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/bestows-Z/dev-hub/backend/internal/project"
	"github.com/bestows-Z/dev-hub/backend/internal/search"
	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/bestows-Z/dev-hub/backend/internal/store"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/redis/go-redis/v9"
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
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password})
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := redisClient.Ping(redisCtx).Err(); err != nil {
		redisCancel()
		logger.Fatal("initialize redis failed", zap.Error(err))
	}
	redisCancel()
	defer redisClient.Close()
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
	searchClient := search.New(cfg.Search.URL)
	searchCtx, searchCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := searchClient.Rebuild(searchCtx, postgresClient.DB); err != nil {
		logger.Warn("article search index unavailable; using database search", zap.Error(err))
	} else {
		logger.Info("article search index ready", zap.String("elasticsearch", searchClient.URL()))
	}
	searchCancel()
	contentHandler := content.NewHandler(content.NewSearchRepository(postgresClient.DB, searchClient), logger)
	storeHandler := store.NewHandler(store.NewRepository(postgresClient.DB), logger)
	projectHandler := project.NewHandler(project.NewRepository(postgresClient.DB), logger)
	objectStore, err := storage.New(cfg.Storage)
	if err != nil {
		logger.Fatal("initialize project storage failed", zap.Error(err))
	}
	bucketCtx, bucketCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := objectStore.EnsureBucket(bucketCtx); err != nil {
		bucketCancel()
		logger.Fatal("prepare project storage bucket failed", zap.Error(err))
	}
	bucketCancel()
	authHandler := auth.NewHandler(userRepository, auth.NewTokens(cfg.Auth.JWTSecret), postgresClient.DB, objectStore, logger)
	previewHandler := project.NewPreviewHandler(postgresClient.DB, objectStore, logger)
	galleryHandler := gallery.NewHandler(postgresClient.DB, objectStore, logger)
	runtimeHandler := project.NewRuntimeHandler(postgresClient.DB, logger)
	adminHandler := admin.NewHandler(postgresClient.DB, objectStore, logger)
	adminHandler.SetArticleIndex(searchClient)
	assistantHandler := assistant.NewHandler(postgresClient.DB, cfg.Assistant, logger)
	assistantHandler.SetRateLimiter(assistant.NewRedisRateLimiter(redisClient))
	engagementHandler := engagement.NewHandler(postgresClient.DB, logger)
	analyticsCtx, stopAnalytics := context.WithCancel(context.Background())
	defer stopAnalytics()
	var analyticsStore *analytics.Store
	var analyticsBroker *analytics.Broker
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	analyticsStore, err = analytics.NewStore(setupCtx, cfg.Analytics)
	setupCancel()
	if err != nil {
		logger.Warn("visitor analytics storage unavailable", zap.Error(err))
	} else {
		defer func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = analyticsStore.Close(closeCtx)
		}()
		analyticsBroker, err = analytics.NewBroker(cfg.Analytics, logger)
		if err != nil {
			logger.Warn("visitor analytics queue unavailable", zap.Error(err))
		} else {
			defer analyticsBroker.Close()
			go analyticsBroker.RunPublisher(analyticsCtx)
			go func() {
				if err := analyticsBroker.RunConsumer(analyticsCtx, analyticsStore); err != nil {
					logger.Warn("visitor analytics worker stopped", zap.Error(err))
				}
			}()
		}
	}
	analyticsHandler := analytics.NewHandler(analyticsStore, logger)
	mediaHandler := media.NewHandler(postgresClient.DB, objectStore, logger)
	router := httprouter.New(postgresClient.SQLDB, userHandler, authHandler, contentHandler, storeHandler, projectHandler, previewHandler, runtimeHandler, galleryHandler, mediaHandler, adminHandler, assistantHandler, engagementHandler, analyticsHandler, analyticsBroker)
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
	stopAnalytics()
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
