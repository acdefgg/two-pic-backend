package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sync-photo-backend/internal/config"
	"sync-photo-backend/internal/handlers"
	"sync-photo-backend/internal/middleware"
	"sync-photo-backend/internal/repository"
	"sync-photo-backend/internal/services"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Run() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Setup logger
	setupLogger(cfg.Log.Level)

	// Connect to database
	db, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}
	log.Info().Msg("Database connection established")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	pairRepo := repository.NewPairRepository(db)
	photoRepo := repository.NewPhotoRepository(db)

	// Initialize services
	userService := services.NewUserService(userRepo, cfg.JWT.Secret)
	pairService := services.NewPairService(pairRepo, userRepo)
	photoService, err := services.NewPhotoService(
		photoRepo,
		pairRepo,
		cfg.AWS.Region,
		cfg.AWS.S3Bucket,
		cfg.AWS.AccessKey,
		cfg.AWS.SecretKey,
		cfg.AWS.Endpoint,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create photo service")
	}
	wsHub := services.NewWSHub(pairService)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService)
	pairHandler := handlers.NewPairHandler(pairService, wsHub)
	photoHandler := handlers.NewPhotoHandler(photoService)
	wsHandler := handlers.NewWebSocketHandler(wsHub, userService, pairService, photoService)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(corsMiddleware)

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Post("/users", userHandler.CreateUser)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(userService))
			r.Post("/pairs", pairHandler.CreatePair)
			r.Delete("/pairs/{pair_id}", pairHandler.DeletePair)
			r.Get("/photos", photoHandler.GetPhotos)
			r.Post("/photos/upload", photoHandler.UploadPhoto)
		})
	})

	// WebSocket route
	r.Get("/ws", wsHandler.HandleWebSocket)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().
			Str("host", cfg.Server.Host).
			Int("port", cfg.Server.Port).
			Msg("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Close WebSocket connections
	// Note: wsHub doesn't have a Close method, connections will be closed when server shuts down

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

// setupLogger configures zerolog logger
func setupLogger(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// corsMiddleware handles CORS
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
