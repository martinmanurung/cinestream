package main

import (
	"context"
	"fmt"
	"log"

	"github.com/martinmanurung/cinestream/internal/domain/movies/repository"
	"github.com/martinmanurung/cinestream/internal/platform/queue"
	"github.com/martinmanurung/cinestream/internal/platform/transcoding"
	"gorm.io/gorm"
)

// JobProcessor handles transcoding job processing
type JobProcessor struct {
	db                 *gorm.DB
	queueService       queue.QueueService
	transcodingService transcoding.TranscodingService
	movieRepo          *repository.MovieRepository
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(
	db *gorm.DB,
	queueService queue.QueueService,
	transcodingService transcoding.TranscodingService,
	movieRepo *repository.MovieRepository,
) *JobProcessor {
	return &JobProcessor{
		db:                 db,
		queueService:       queueService,
		transcodingService: transcodingService,
		movieRepo:          movieRepo,
	}
}

// Start begins processing jobs from the queue
func (p *JobProcessor) Start(ctx context.Context) error {
	log.Println("Job processor started, waiting for transcoding jobs...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Job processor stopped")
			return ctx.Err()
		default:
			// Consume job from queue (blocking call with timeout)
			job, err := p.queueService.ConsumeTranscodingJob(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("Error consuming job: %v", err)
				continue
			}

			if job == nil {
				// No job available (timeout), continue
				continue
			}

			// Process the job
			log.Printf("Processing job for movie ID: %d", job.MovieID)
			if err := p.processJob(ctx, job); err != nil {
				log.Printf("Error processing job for movie %d: %v", job.MovieID, err)
			}
		}
	}
}

// processJob handles the transcoding of a single movie
func (p *JobProcessor) processJob(ctx context.Context, job *queue.TranscodingJob) error {
	movieID := job.MovieID
	rawFilePath := job.RawFilePath

	// Update status to PROCESSING
	log.Printf("Movie %d: Updating status to PROCESSING", movieID)
	if err := p.movieRepo.UpdateMovieVideo(ctx, movieID, map[string]interface{}{
		"upload_status": "PROCESSING",
	}); err != nil {
		return fmt.Errorf("failed to update status to PROCESSING: %w", err)
	}

	// Perform transcoding
	log.Printf("Movie %d: Starting transcoding from %s", movieID, rawFilePath)
	hlsURL, err := p.transcodingService.TranscodeToHLS(ctx, movieID, rawFilePath)
	if err != nil {
		// Update status to FAILED with error message
		log.Printf("Movie %d: Transcoding FAILED: %v", movieID, err)
		updateErr := p.movieRepo.UpdateMovieVideo(ctx, movieID, map[string]interface{}{
			"upload_status": "FAILED",
			"error_message": err.Error(),
		})
		if updateErr != nil {
			log.Printf("Movie %d: Failed to update error status: %v", movieID, updateErr)
		}
		return fmt.Errorf("transcoding failed: %w", err)
	}

	// Update status to READY with HLS URL
	log.Printf("Movie %d: Transcoding completed successfully, HLS URL: %s", movieID, hlsURL)
	if err := p.movieRepo.UpdateMovieVideo(ctx, movieID, map[string]interface{}{
		"upload_status":    "READY",
		"hls_playlist_url": hlsURL,
		"error_message":    nil,
	}); err != nil {
		return fmt.Errorf("failed to update status to READY: %w", err)
	}

	log.Printf("Movie %d: Processing completed successfully", movieID)
	return nil
}
