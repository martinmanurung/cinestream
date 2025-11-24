package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/martinmanurung/cinestream/internal/platform/config" // Sesuaikan path ini
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Initialize minio
func InitMinIO(cfg config.MinIOConfig) (*minio.Client, error) {
	// 1. Init minio client
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing minio client: %w", err)
	}

	// 2. trying to connect minio
	if _, err := minioClient.ListBuckets(context.Background()); err != nil {
		return nil, fmt.Errorf("error verifying minio connection: %w", err)
	}

	// 3. make sure the bucket available
	// This is an 'idempotent' function, safe to run multiple times
	err = checkAndCreateBucket(minioClient, cfg.BucketRaw, false)
	if err != nil {
		return nil, err
	}

	// Set bucket 'processed' to public-read
	err = checkAndCreateBucket(minioClient, cfg.BucketProcessed, true)
	if err != nil {
		return nil, err
	}

	return minioClient, nil
}

// helper function to create bucket if not ready
func checkAndCreateBucket(client *minio.Client, bucketName string, isPublic bool) error {
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("error checking bucket '%s': %w", bucketName, err)
	}

	if !exists {
		// Create the bucket if it doesn't exist
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("error creating bucket '%s': %w", bucketName, err)
		}
		log.Printf("Bucket '%s' created successfully.", bucketName)
	}

	// If the bucket is 'processed', set it to public-read for HLS
	if isPublic {
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::%s/*"]
				}
			]
		}`, bucketName)

		err = client.SetBucketPolicy(ctx, bucketName, policy)
		if err != nil {
			return fmt.Errorf("error setting policy public-read for bucket '%s': %w", bucketName, err)
		}
	}
	return nil
}
