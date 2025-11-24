package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	movieRepository "github.com/martinmanurung/cinestream/internal/domain/movies/repository"
	"github.com/martinmanurung/cinestream/internal/platform/config"
	"github.com/martinmanurung/cinestream/internal/platform/database"
	"github.com/martinmanurung/cinestream/internal/platform/queue"
	storage "github.com/martinmanurung/cinestream/internal/platform/strorage"
	"github.com/martinmanurung/cinestream/internal/platform/transcoding"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("Starting CineStream Transcoding Worker...")

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
	log.Println("MinIO initialized successfully")

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
	log.Println("Redis initialized successfully")

	// Initialize services
	queueService := queue.NewRedisQueue(redisClient)
	transcodingService := transcoding.NewTranscodingService(minioClient, cfg.MinIO.BucketRaw, cfg.MinIO.BucketProcessed)

	// Initialize repository
	movieRepo := movieRepository.NewMovieRepository(db)

	// Create job processor
	processor := NewJobProcessor(db, queueService, transcodingService, movieRepo)

	// Create context with cancellation for graceful shutdown
	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start processing jobs in a goroutine
	processorDone := make(chan error, 1)
	go func() {
		processorDone <- processor.Start(workerCtx)
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Received shutdown signal, stopping worker...")
		cancel()        // Cancel the processor context
		<-processorDone // Wait for processor to finish current job
		log.Println("Worker stopped gracefully")
	case err := <-processorDone:
		log.Fatalf("Worker stopped with error: %v", err)
	}
}
