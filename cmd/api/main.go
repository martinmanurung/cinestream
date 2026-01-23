package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	movieDelivery "github.com/martinmanurung/cinestream/internal/domain/movies/delivery"
	movieRepository "github.com/martinmanurung/cinestream/internal/domain/movies/repository"
	movieUsecase "github.com/martinmanurung/cinestream/internal/domain/movies/usecase"
	orderDelivery "github.com/martinmanurung/cinestream/internal/domain/orders/delivery"
	orderRepository "github.com/martinmanurung/cinestream/internal/domain/orders/repository"
	orderUsecase "github.com/martinmanurung/cinestream/internal/domain/orders/usecase"
	"github.com/martinmanurung/cinestream/internal/domain/users/delivery"
	"github.com/martinmanurung/cinestream/internal/domain/users/repository"
	"github.com/martinmanurung/cinestream/internal/domain/users/usecase"
	"github.com/martinmanurung/cinestream/internal/platform/config"
	"github.com/martinmanurung/cinestream/internal/platform/database"
	"github.com/martinmanurung/cinestream/internal/platform/payment"
	"github.com/martinmanurung/cinestream/internal/platform/queue"
	storage "github.com/martinmanurung/cinestream/internal/platform/strorage"
	"github.com/martinmanurung/cinestream/pkg/jwt"
	"github.com/martinmanurung/cinestream/pkg/middleware"
	customValidator "github.com/martinmanurung/cinestream/pkg/validator"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	// Setup zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	zlog.Info().Msg("Starting CineStream API Server...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}
	defer sqlDB.Close()

	ctx := context.Background()

	// Initialize MinIO
	minioClient, err := storage.InitMinIO(cfg.MinIO)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO: %v", err)
	}
	zlog.Info().Msg("MinIO initialized successfully")

	// Initialize Redis client
	redisAddr := cfg.Redis.Host + ":" + cfg.Redis.Port
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Ping Redis to verify connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	zlog.Info().Msg("Redis initialized successfully")

	// Initialize services
	storageService := storage.NewStorageService(minioClient, cfg.MinIO.BucketRaw, cfg.MinIO.BucketProcessed)
	queueService := queue.NewRedisQueue(redisClient)

	// Initialize Echo
	e := echo.New()
	e.Use(middleware.RequestID())
	e.HideBanner = false

	// Register validator
	e.Validator = customValidator.New()

	// Initialize JWT service
	jwtService := jwt.NewJWTService(cfg.JWT.SecretKey)

	// Initialize repositories
	userRepo := repository.NewUser(db)
	movieRepo := movieRepository.NewMovieRepository(db)
	orderRepo := orderRepository.NewOrderRepository(db)

	// Create adapters for order usecase
	movieRepoAdapter := orderRepository.NewMovieRepositoryAdapter(movieRepo)
	userRepoAdapter := orderRepository.NewUserRepositoryAdapter(userRepo)

	// Initialize payment service
	paymentService := payment.NewMidtransService(
		cfg.PaymentGW.ServerKey,
		cfg.PaymentGW.ClientKey,
		cfg.PaymentGW.IsProduction,
	)

	// Initialize use cases
	userUsecase := usecase.NewUsecase(userRepo, jwtService)
	movieUsecaseInstance := movieUsecase.NewMovieUsecase(movieRepo, storageService, queueService)
	orderUsecaseInstance := orderUsecase.NewOrderUsecase(orderRepo, movieRepoAdapter, userRepoAdapter, paymentService)

	// Initialize handlers
	userHandler := delivery.NewHandler(ctx, userUsecase)
	movieHandler := movieDelivery.NewMovieHandler(ctx, movieUsecaseInstance)
	genreHandler := movieDelivery.NewGenreHandler(ctx, movieUsecaseInstance)
	orderHandler := orderDelivery.NewOrderHandler(ctx, orderUsecaseInstance)
	webhookHandler := orderDelivery.NewWebhookHandler(ctx, orderRepo, paymentService, cfg.PaymentGW.ServerKey)
	streamingHandler := orderDelivery.NewStreamingHandler(ctx, orderUsecaseInstance)

	// Setup routes
	setupRoutes(e, userHandler, movieHandler, genreHandler, orderHandler, webhookHandler, streamingHandler, jwtService)

	// Start server in goroutine
	go func() {
		port := cfg.Server.Port
		if port == "" {
			port = "8080"
		}

		zlog.Info().Str("port", port).Msg("Starting HTTP server")
		if err := e.Start(":" + port); err != nil {
			zlog.Info().Err(err).Msg("Server stopped")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	zlog.Info().Msg("Shutting down server...")

	// Gracefully shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	zlog.Info().Msg("Server exited successfully")
}
