package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type cacheRepository struct {
	cacheSource CacheSource
}

func NewCacheRepository(cacheSource CacheSource) *cacheRepository {
	return &cacheRepository{
		cacheSource: cacheSource,
	}
}

func (c *cacheRepository) LockEventByID(ctx context.Context, eventID uuid.UUID) (bool, error) {
	ok, err := c.cacheSource.LockEventByID(ctx, eventID)
	if err != nil {
		return false, fmt.Errorf("cacheRepository.LockEventByID: %w", err)
	}
	return ok, nil
}

func (c *cacheRepository) DeleteEventByID(ctx context.Context, eventID uuid.UUID) error {
	if err := c.cacheSource.DeleteEventByID(ctx, eventID); err != nil {
		return fmt.Errorf("cacheRepository.DeleteEventByID: %w", err)
	}
	return nil
}
