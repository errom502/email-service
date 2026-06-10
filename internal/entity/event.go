package entity

import (
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

const (
	providerTypeEmail = "email"
)

type VerificationCreatedEvent struct {
	EventID        uuid.UUID `json:"event_id"`
	VerificationID uuid.UUID `json:"verification_id"`
	UserExtID      string    `json:"user_ext_id"`
	ProviderType   string    `json:"provider_type"`
	TargetValue    string    `json:"target_value"`
	Secret         string    `json:"secret"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func (v *VerificationCreatedEvent) Validate() error {
	if v == nil {
		return nil
	}

	if v.EventID == uuid.Nil {
		return errors.New("VerificationCreatedEvent.Validate: event_id is empty")
	}

	if v.VerificationID == uuid.Nil {
		return errors.New("VerificationCreatedEvent.Validate: verification_id is empty")
	}

	if v.TargetValue == "" {
		return errors.New("VerificationCreatedEvent.Validate: target_value is empty")
	}

	if v.Secret == "" {
		return errors.New("VerificationCreatedEvent.Validate: secret is empty")
	}

	if v.ProviderType != providerTypeEmail {
		return fmt.Errorf("VerificationCreatedEvent.Validate: invalid provider type: %s", v.ProviderType)
	}

	if _, err := mail.ParseAddress(v.TargetValue); err != nil {
		return fmt.Errorf("VerificationCreatedEvent.Validate: invalid target value: %s", v.TargetValue)
	}

	return nil
}

type DeliveryFailedEvent struct {
	EventID        uuid.UUID `json:"event_id"`
	VerificationID uuid.UUID `json:"verification_id"`
	Reason         string    `json:"reason"`
}
