package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/errom502/email-service/internal/entity"
	"github.com/errom502/email-service/internal/tools"
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
