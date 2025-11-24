package queue

import (
	"context"
	"fmt"

	"github.com/martinmanurung/cinestream/internal/platform/config" // Sesuaikan path ini
	"github.com/redis/go-redis/v9"
)

// InitRedis menginisialisasi koneksi ke server Redis
func InitRedis(cfg config.RedisConfig) (*redis.Client, error) {

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		client.Close() // Tutup jika gagal
		return nil, fmt.Errorf("error verifying Redis connection: %w", err)
	}

	return client, nil
}
