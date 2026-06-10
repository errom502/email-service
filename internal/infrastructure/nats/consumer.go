package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/errom502/email-service/internal/entity"
	"github.com/errom502/email-service/internal/tools"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type natsConsumer struct {
	js             jetstream.JetStream
	consumerConfig jetstream.ConsumerConfig
	streamName     string
	handler        VerificationCreatedHandler
	fetchRetryWait time.Duration
	logger         tools.Logger
}

func NewNatsConsumer(
	js jetstream.JetStream,
	consumerConfig jetstream.ConsumerConfig,
	streamName string,
	handler VerificationCreatedHandler,
	fetchRetryWait time.Duration,
	logger tools.Logger,
) *natsConsumer {
	return &natsConsumer{
		js:             js,
		consumerConfig: consumerConfig,
		streamName:     streamName,
		handler:        handler,
		fetchRetryWait: fetchRetryWait,
		logger:         logger,
	}
}

// Run - пытается найти стрим в инфре, создается или обновляется консьюмер в стриме, читаются события
func (c *natsConsumer) Run(appCtx context.Context) error {
	ctx, cancel := context.WithCancel(appCtx)
	defer cancel()

	stream, err := c.js.Stream(
		ctx,
		c.streamName,
	)
	if err != nil {
		return fmt.Errorf("natsConsumer.Run: can't get stream: %w", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(
		ctx,
		c.consumerConfig,
	)

	if err != nil {
		return fmt.Errorf("natsConsumer.Run: can't create consumer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-appCtx.Done():
			return nil
		default:
		}

		// TODO: в будущей реализации рассмотреть получение нескольких событий : fetch(10)
		msgs, err := consumer.Fetch(1)
		if err != nil {
			c.logger.Error("natsConsumer.Run: can't fetch message", zap.Error(err))

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(c.fetchRetryWait):
			}
			continue
		}

		for msg := range msgs.Messages() {
			c.handleMessage(ctx, msg)
		}
	}
}

func (c *natsConsumer) handleMessage(
	ctx context.Context,
	msg jetstream.Msg,
) {
	var event entity.VerificationCreatedEvent

	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		c.logger.Error(
			"natsConsumer.handleMessage: invalid message",
			zap.ByteString("msg data", msg.Data()),
			zap.Error(err),
		)
		if ackErr := msg.Ack(); ackErr != nil {
			c.logger.Error("natsConsumer.handleMessage: failed to ack message", zap.Error(ackErr))
		}
		return
	}

	result := c.handler.HandleVerificationCreated(ctx, &event)

	switch result {
	case entity.ResultAck:
		if ackErr := msg.Ack(); ackErr != nil {
			c.logger.Error("natsConsumer.handleMessage: failed to ack message", zap.Error(ackErr))
		}
	case entity.ResultRetry:
		if nakErr := msg.Nak(); nakErr != nil {
			c.logger.Error("natsConsumer.handleMessage: failed to nak message", zap.Error(nakErr))
		}
	default:
		c.logger.Error("natsConsumer.handleMessage: unknown handle result", zap.Any("result", result))
		if nakErr := msg.Nak(); nakErr != nil {
			c.logger.Error("natsConsumer.handleMessage: failed to nak message", zap.Error(nakErr))
		}
	}
}
