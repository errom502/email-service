package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/errom502/email-service/internal/entity"
	"github.com/errom502/email-service/internal/shared"
	smtpSource "github.com/errom502/email-service/internal/source/smtp"
	"github.com/errom502/email-service/internal/tools"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type eventType string

const (
	verificationCreatedEvent eventType = "verification.email.created"
)

type emailUsecase struct {
	cache          CacheRepository
	smtp           SmtpRepository
	linkTool       LinkTool
	hashTool       HashTool
	messageTool    MessageTool
	eventIDTool    EventIDTool
	eventPublisher EventPublisher
	logger         tools.Logger
}

func NewEmailUsecase(
	cache CacheRepository,
	smtp SmtpRepository,
	linkTool LinkTool,
	hashTool HashTool,
	messageTool MessageTool,
	eventIDTool EventIDTool,
	eventPublisher EventPublisher,
	logger tools.Logger,
) *emailUsecase {
	return &emailUsecase{
		cache:          cache,
		smtp:           smtp,
		linkTool:       linkTool,
		hashTool:       hashTool,
		messageTool:    messageTool,
		eventIDTool:    eventIDTool,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

func (e *emailUsecase) HandleVerificationCreated(
	ctx context.Context,
	event *entity.VerificationCreatedEvent,
) entity.HandleResult {
	if event == nil {
		e.logger.Error(
			"emailUsecase.HandleVerificationCreated: empty event",
			zap.Error(ErrEmptyEvent),
		)
		return entity.ResultAck
	}

	locked, err := e.cache.LockEventByID(ctx, event.EventID)
	if err != nil {
		e.logger.Error(
			"emailUsecase.HandleVerificationCreated: failed to lock event",
			zap.Error(err),
		)
		return entity.ResultRetry
	}

	// Если ключ уже создан, то событие нужно завершить
	if !locked {
		return entity.ResultAck
	}

	if err := event.Validate(); err != nil {
		e.logger.Error(
			"emailUsecase.HandleVerificationCreated: validation failed",
			zap.String("eventID", event.EventID.String()),
			zap.String("verificationID", event.VerificationID.String()),
			zap.Error(err),
		)
		return e.publishDeliveryFailed(
			ctx,
			err.Error(),
			event.EventID,
			event.VerificationID,
			verificationCreatedEvent,
		)
	}

	if event.ExpiresAt.Before(time.Now()) {
		e.logger.Error(
			"emailUsecase.HandleVerificationCreated: verification expired",
			zap.String("eventID", event.EventID.String()),
			zap.String("verificationID", event.VerificationID.String()),
			zap.Error(ErrVerificationExpired),
		)
		return entity.ResultAck
	}

	hash := e.hashTool.BuildVerificationHash(
		event.VerificationID,
		event.Secret,
	)

	link := e.linkTool.BuildVerificationLink(
		event.VerificationID,
		hash,
	)

	message := e.messageTool.BuildMessage(
		link,
	)

	if err := e.smtp.Send(
		ctx,
		event.TargetValue,
		message,
	); err != nil {
		switch {
		case errors.Is(err, shared.ErrPermanentSMTP):
			fallthrough
		case errors.Is(err, smtpSource.ErrBuildMessage):
			fallthrough
		case errors.Is(err, smtpSource.ErrInvalidTarget):
			e.logger.Error(
				"emailUsecase.HandleVerificationCreated: failed to send message to smtp",
				zap.Error(err),
			)
			return e.publishDeliveryFailed(
				ctx,
				err.Error(),
				event.EventID,
				event.VerificationID,
				verificationCreatedEvent,
			)
		case errors.Is(err, shared.ErrTemporarySMTP):
			e.logger.Error(
				"emailUsecase.HandleVerificationCreated: failed to send message to smtp",
				zap.Error(err),
			)
			if cacheErr := e.cache.DeleteEventByID(ctx, event.EventID); cacheErr != nil {
				e.logger.Error(
					"emailUsecase.publishDeliveryFailed: failed delete redis key",
					zap.Error(cacheErr),
				)
			}
			return entity.ResultRetry
		default:
			e.logger.Error(
				"emailUsecase.HandleVerificationCreated: unknown error",
				zap.Error(err),
			)
			if cacheErr := e.cache.DeleteEventByID(ctx, event.EventID); cacheErr != nil {
				e.logger.Error(
					"emailUsecase.publishDeliveryFailed: failed delete redis key",
					zap.Error(cacheErr),
				)
			}
			return entity.ResultRetry
		}
	}

	return entity.ResultAck
}

// publishDeliveryFailed отправляет DeliveryFailedEvent в брокер.
//
// При успехе возвращает ResultAck.
// В случае неудачной отправки события deliveryFailed, происходит попытка удаления ключа из кеша, вернется ReasultRetry
func (e *emailUsecase) publishDeliveryFailed(ctx context.Context, reason string, eventID, verificationID uuid.UUID, eventType eventType) entity.HandleResult {
	err := e.eventPublisher.PublishDeliveryFailed(
		ctx,
		&entity.DeliveryFailedEvent{
			EventID:        e.eventIDTool.BuildDerivedID(eventID, string(eventType)),
			VerificationID: verificationID,
			Reason:         reason,
		},
	)

	if err == nil {
		return entity.ResultAck
	}

	e.logger.Error(
		"emailUsecase.publishDeliveryFailed: failed to publish DeliveryFailedEvent",
		zap.String("eventID", eventID.String()),
		zap.String("verificationID", verificationID.String()),
		zap.Error(err),
	)

	if cacheErr := e.cache.DeleteEventByID(ctx, eventID); cacheErr != nil {
		e.logger.Error(
			"emailUsecase.publishDeliveryFailed: failed delete redis key after publish delivery failed event",
			zap.String("eventID", eventID.String()),
			zap.Error(cacheErr),
		)
	}

	return entity.ResultRetry
}
