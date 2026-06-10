package nats

import (
	"context"
	"email-service/internal/entity"
	"email-service/internal/tools"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

type producer struct {
	js      jetstream.JetStream
	subject string
	logger  tools.Logger
}

func NewProducer(
	js jetstream.JetStream,
	subject string,
	logger tools.Logger,
) *producer {
	return &producer{
		js:      js,
		subject: subject,
		logger:  logger,
	}
}

func (p *producer) PublishDeliveryFailed(ctx context.Context, event *entity.DeliveryFailedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("producer.PublishDeliveryFailed: failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(ctx, p.subject, data)
	if err != nil {
		return fmt.Errorf("producer.PublishDeliveryFailed: failed to publish event: %w", err)
	}

	return nil
}
