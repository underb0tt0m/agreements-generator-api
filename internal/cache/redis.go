package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:generate mockgen -source=redis.go -destination=../mocks/redis.go -package=mocks
type Cacher interface {
	SetJobStatus(ctx context.Context, jobID string, status string) error
	GetJobStatus(ctx context.Context, jobID string) (string, error)
	DeleteJobStatus(ctx context.Context, jobID string) error
	Close() error
}

type redisCacher struct {
	client       *redis.Client
	jobStatusTTL time.Duration
}

func New(host, port, password string, db int, jobStatusTTL time.Duration) Cacher {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})
	return &redisCacher{
		client:       client,
		jobStatusTTL: jobStatusTTL,
	}
}

func (r *redisCacher) SetJobStatus(ctx context.Context, jobID, status string) error {
	return r.client.Set(ctx, "job:"+jobID, status, r.jobStatusTTL).Err()
}

func (r *redisCacher) GetJobStatus(ctx context.Context, jobID string) (string, error) {
	return r.client.Get(ctx, "job:"+jobID).Result()
}

func (r *redisCacher) DeleteJobStatus(ctx context.Context, jobID string) error {
	return r.client.Del(ctx, "job:"+jobID).Err()
}
func (r *redisCacher) Close() error {
	return r.client.Close()
}
