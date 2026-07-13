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
	"github.com/madebyduy/food-social/internal/module/geo"
	"github.com/madebyduy/food-social/internal/module/governance"
	"github.com/madebyduy/food-social/internal/module/location"
	"github.com/madebyduy/food-social/internal/module/media"
	"github.com/madebyduy/food-social/internal/module/place"
	"github.com/madebyduy/food-social/internal/module/platform"
	"github.com/madebyduy/food-social/internal/module/post"
	"github.com/madebyduy/food-social/internal/module/search"
	"github.com/madebyduy/food-social/internal/module/social"
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

	postRepo := post.NewRepository(db)
	postSvc := post.NewService(db, postRepo, logger)
	postHandler := post.NewHandler(postSvc)

	geoRepo := geo.NewRepository(db)
	geoSvc := geo.NewService(db, geoRepo, logger)
	geoHandler := geo.NewHandler(geoSvc)

	placeRepo := place.NewRepository(db)
	placeSvc := place.NewService(db, placeRepo, logger)
	placeHandler := place.NewHandler(placeSvc)

	if !cfg.Cloudinary.Configured() {
		logger.Warn("cloudinary chưa cấu hình; các endpoint /media sẽ lỗi khi gọi")
	}
	mediaStorage := media.NewCloudinaryStorage(
		cfg.Cloudinary.CloudName, cfg.Cloudinary.APIKey, cfg.Cloudinary.APISecret, cfg.Cloudinary.Folder,
	)
	mediaRepo := media.NewRepository(db)
	mediaSvc := media.NewService(db, mediaRepo, mediaStorage, logger)
	mediaHandler := media.NewHandler(mediaSvc)
	socialHandler := social.NewHandler(social.NewService(db))
	governanceHandler := governance.NewHandler(governance.NewService(db))
	searchHandler := search.NewHandler(search.NewService(db))
	locationHandler := location.NewHandler(location.NewService(db))

	mux := router.New(router.Dependencies{
		DB:                db,
		AuthHandler:       authHandler,
		UserHandler:       userHandler,
		PostHandler:       postHandler,
		GeoHandler:        geoHandler,
		PlaceHandler:      placeHandler,
		MediaHandler:      mediaHandler,
		SocialHandler:     socialHandler,
		GovernanceHandler: governanceHandler,
		SearchHandler:     searchHandler,
		LocationHandler:   locationHandler,
		SessionResolver:   authSvc,
	})

	rateLimiter := platform.NewRateLimiter(50, 100, nil)
	stopCleanup := rateLimiter.StartCleanup(time.Minute)
	defer stopCleanup()

	handler := middleware.Chain(mux,
		middleware.Recovery(logger),
		middleware.CORS(),
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
