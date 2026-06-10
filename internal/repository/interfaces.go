package repository

import (
	"context"

	"github.com/google/uuid"
)

type CacheSource interface {
	LockEventByID(ctx context.Context, eventID uuid.UUID) (bool, error)
	DeleteEventByID(ctx context.Context, eventID uuid.UUID) error
}

type SmtpSource interface {
	Send(
		ctx context.Context,
		target string,
		body string,
	) error
}
