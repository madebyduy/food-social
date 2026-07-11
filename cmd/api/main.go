package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/madebyduy/food-social/internal/auth"
	"github.com/madebyduy/food-social/internal/config"
	"github.com/madebyduy/food-social/internal/database"
	"github.com/madebyduy/food-social/internal/middleware"
	"github.com/madebyduy/food-social/internal/module/platform"
	"github.com/madebyduy/food-social/internal/module/user"
	"github.com/madebyduy/food-social/internal/router"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("could not load .env; ignoring for non-local environments", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(db, authRepo, cfg.SessionTTL, nil, logger)
	authHandler := auth.NewHandler(authSvc)

	userRepo := user.NewRepository(db)
	userSvc := user.NewService(db, userRepo, logger)
	userHandler := user.NewHandler(userSvc)

	mux := router.New(router.Dependencies{
		DB:              db,
		AuthHandler:     authHandler,
		UserHandler:     userHandler,
		SessionResolver: authSvc,
	})

	rateLimiter := platform.NewRateLimiter(50, 100, nil)
	stopCleanup := rateLimiter.StartCleanup(time.Minute)
	defer stopCleanup()

	handler := middleware.Chain(mux,
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.Logging(logger),
		middleware.RateLimit(rateLimiter),
	)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	runWithGracefulShutdown(srv, logger)
}

func runWithGracefulShutdown(srv *http.Server, logger *slog.Logger) {
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	logger.Info("server started", "addr", srv.Addr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
	logger.Info("server stopped")
}
