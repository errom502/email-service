package usecase

import (
	"context"
	"email-service/internal/entity"

	"github.com/google/uuid"
)

type CacheRepository interface {
	// LockEventByID атомарно создает ключ идемпотентности для события.
	// Возвращает true, если ключ создан, false если ключ уже существует.
	LockEventByID(ctx context.Context, eventID uuid.UUID) (bool, error)
	// DeleteEventByID удаляет ключ идемпотентности для события.
	DeleteEventByID(ctx context.Context, eventID uuid.UUID) error
}

type SmtpRepository interface {
	Send(
		ctx context.Context,
		target string,
		body string,
	) error
}

type LinkTool interface {
	BuildVerificationLink(
		verificationID uuid.UUID,
		hash string,
	) string
}

type HashTool interface {
	BuildVerificationHash(
		verificationID uuid.UUID,
		secret string,
	) string
}

type EventIDTool interface {
	BuildDerivedID(
		sourceEventID uuid.UUID,
		eventType string,
	) uuid.UUID
}

type MessageTool interface {
	BuildMessage(verificationLink string) string
}

type EventPublisher interface {
	PublishDeliveryFailed(ctx context.Context, event *entity.DeliveryFailedEvent) error
}
