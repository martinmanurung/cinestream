package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/minio/minio-go/v7"
)

type StorageService struct {
	client          *minio.Client
	bucketRaw       string
	bucketProcessed string
}

func NewStorageService(client *minio.Client, bucketRaw, bucketProcessed string) *StorageService {
	return &StorageService{
		client:          client,
		bucketRaw:       bucketRaw,
		bucketProcessed: bucketProcessed,
	}
}

// UploadRawVideo uploads a video file to the raw bucket
func (s *StorageService) UploadRawVideo(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader, movieID int64) (string, error) {
	// Generate object name: raw-videos/movie-{id}.ext
	ext := filepath.Ext(fileHeader.Filename)
	objectName := fmt.Sprintf("raw-videos/movie-%d%s", movieID, ext)

	// Upload to MinIO
	_, err := s.client.PutObject(
		ctx,
		s.bucketRaw,
		objectName,
		file,
		fileHeader.Size,
		minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("Content-Type"),
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload video to MinIO: %w", err)
	}

	return objectName, nil
}

// GetRawVideoURL returns the internal URL for raw video (for worker processing)
func (s *StorageService) GetRawVideoURL(objectName string) string {
	return fmt.Sprintf("%s/%s", s.bucketRaw, objectName)
}

// GetHLSURL returns the public URL for HLS playlist
func (s *StorageService) GetHLSURL(ctx context.Context, movieID int64) (string, error) {
	objectName := fmt.Sprintf("processed-videos/%d/master.m3u8", movieID)

	// Check if object exists
	_, err := s.client.StatObject(ctx, s.bucketProcessed, objectName, minio.StatObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("HLS playlist not found: %w", err)
	}

	// Return public URL (assuming bucket is public-read)
	// Format: http://minio-endpoint/bucket/object-path
	url := fmt.Sprintf("http://%s/%s/%s", s.client.EndpointURL().Host, s.bucketProcessed, objectName)
	return url, nil
}

// DeleteRawVideo deletes a raw video file
func (s *StorageService) DeleteRawVideo(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucketRaw, objectName, minio.RemoveObjectOptions{})
}

// DeleteProcessedVideo deletes all processed video files for a movie
func (s *StorageService) DeleteProcessedVideo(ctx context.Context, movieID int64) error {
	prefix := fmt.Sprintf("processed-videos/%d/", movieID)

	// List all objects with the prefix
	objectsCh := s.client.ListObjects(ctx, s.bucketProcessed, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	// Delete each object
	for object := range objectsCh {
		if object.Err != nil {
			return object.Err
		}
		err := s.client.RemoveObject(ctx, s.bucketProcessed, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// StreamFile streams a file from MinIO
func (s *StorageService) StreamFile(ctx context.Context, bucket, objectName string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	return object, nil
}
