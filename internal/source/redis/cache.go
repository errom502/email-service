package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TODO: переделать *redis.Client на интерфейс, чтобы потому тесты написаить
type redisCache struct {
	client *redis.Client
	keyTTL time.Duration
}

func NewRedisCache(clientConn *redis.Client, keyTTL time.Duration) *redisCache {
	return &redisCache{
		client: clientConn,
		keyTTL: keyTTL,
	}
}

// LockEventByID атомарно создает ключ идемпотентности для события.
// Возвращает true, если ключ создан, false если ключ уже существует.
func (r *redisCache) LockEventByID(ctx context.Context, eventID uuid.UUID) (bool, error) {
	res, err := r.client.SetArgs(
		ctx,
		fmt.Sprintf("processing:event:%s", eventID.String()),
		1,
		redis.SetArgs{
			Mode: "NX",
			TTL:  r.keyTTL,
		},
	).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}

		return false, fmt.Errorf("redisCache.LockEventByID: redis set args: %w", err)
	}
	return res == "OK", nil
}

// DeleteEventByID удаляет ключ идемпотентности для события.
func (r *redisCache) DeleteEventByID(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.client.Del(
		ctx,
		fmt.Sprintf("processing:event:%s", eventID.String()),
	).Result()

	if err != nil {
		return fmt.Errorf("redisCache.DeleteEventByID: %w", err)
	}

	return nil
}
