package repository

import (
	"context"
	"fmt"
)

type smtpRepository struct {
	source SmtpSource
}

func NewSmtpRepository(source SmtpSource) *smtpRepository {
	return &smtpRepository{
		source: source,
	}
}

func (s *smtpRepository) Send(
	ctx context.Context,
	target string,
	body string,
) error {
	if err := s.source.Send(
		ctx,
		target,
		body,
	); err != nil {
		return fmt.Errorf("smtpRepository.Send: %w", err)
	}
	return nil
}
