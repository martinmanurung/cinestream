package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

// QueueService defines the interface for queue operations
type QueueService interface {
	PublishTranscodingJob(ctx context.Context, movieID int64, rawFilePath string) error
	ConsumeTranscodingJob(ctx context.Context) (*TranscodingJob, error)
}

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(client *redis.Client) *RedisQueue {
	return &RedisQueue{client: client}
}

// TranscodingJob represents a transcoding job message
type TranscodingJob struct {
	MovieID     int64  `json:"movie_id"`
	RawFilePath string `json:"raw_file_path"`
}

// PublishTranscodingJob publishes a transcoding job to Redis queue
func (q *RedisQueue) PublishTranscodingJob(ctx context.Context, movieID int64, rawFilePath string) error {
	job := TranscodingJob{
		MovieID:     movieID,
		RawFilePath: rawFilePath,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Push to Redis list (queue)
	queueName := "transcoding:jobs"
	err = q.client.LPush(ctx, queueName, jobData).Err()
	if err != nil {
		return fmt.Errorf("failed to push job to queue: %w", err)
	}

	log.Printf("Published transcoding job for movie_id=%d to queue", movieID)
	return nil
}

// ConsumeTranscodingJob consumes transcoding jobs from Redis queue (for worker)
func (q *RedisQueue) ConsumeTranscodingJob(ctx context.Context) (*TranscodingJob, error) {
	queueName := "transcoding:jobs"

	// Blocking pop from Redis list
	result, err := q.client.BRPop(ctx, 0, queueName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to pop job from queue: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid queue response")
	}

	jobData := result[1]
	var job TranscodingJob
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}
