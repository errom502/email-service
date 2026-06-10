package nats

import (
	"context"

	"github.com/errom502/email-service/internal/entity"
)

type VerificationCreatedHandler interface {
	HandleVerificationCreated(
		ctx context.Context,
		event *entity.VerificationCreatedEvent,
	) entity.HandleResult
}
