package nats

import (
	"context"
	"email-service/internal/entity"
)

type VerificationCreatedHandler interface {
	HandleVerificationCreated(
		ctx context.Context,
		event *entity.VerificationCreatedEvent,
	) entity.HandleResult
}
