package cache

import (
	"context"
	"errors"
	"time"

	"github.com/itsDrac/e-auc/pkg/utils"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidTTL = errors.New("cache: ttl must be > 0")
)

const (
	TempImageListKey = "temp_image_names"
)

type Cacher interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, val string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Ping(ctx context.Context) error
	Close() error
	AddImageNameToTempList(ctx context.Context, imageName string) error
	RemoveImageNameFromTempList(ctx context.Context, imageName string) error
}

type RedisCache struct {
	client *redis.Client
}

func NewRedisClient(ctx context.Context) (*RedisCache, error) {
	addr := utils.GetEnv("REDIS_ADDR", "localhost:6379")
	if addr == "" {
		return nil, errors.New("REDIS_ADDR is required")
	}
	passwrd := utils.GetEnv("REDIS_PASSWORD", "")
	if passwrd == "" {
		return nil, errors.New("REDIS_PASSWORD is required")
	}
	db := utils.GetIntEnv("REDIS_DB", 0)
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     passwrd,
		DB:           db,
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return &RedisCache{
		client: client,
	}, nil

}

func (r *RedisCache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// cache miss - not an error
		return "", false, nil
	}
	if err != nil {
		// real failure (timeout, connection issue, etc.)
		return "", false, err
	}

	return val, true, nil
}

func (r *RedisCache) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	return r.client.Set(ctx, key, val, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisCache) AddImageNameToTempList(ctx context.Context, imageName string) error {
	return r.client.LPush(ctx, TempImageListKey, imageName).Err()
}

func (r *RedisCache) RemoveImageNameFromTempList(ctx context.Context, imageName string) error {
	return r.client.LRem(ctx, TempImageListKey, 0, imageName).Err()
}