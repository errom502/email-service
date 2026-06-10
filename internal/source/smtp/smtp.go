package smtp

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/errom502/email-service/internal/shared"

	go_mail "github.com/wneessen/go-mail"
)

type smtpSource struct {
	client  *go_mail.Client
	from    string
	subject string
	timeout time.Duration
}

func NewSmtpSource(
	client *go_mail.Client,
	from string,
	subject string,
	timeoutMs int,
) (*smtpSource, error) {
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, fmt.Errorf("invalid from mail: %w", err)
	}

	if strings.TrimSpace(subject) == "" {
		return nil, errors.New("smtp subject is empty")
	}

	if timeoutMs <= 0 {
		return nil, errors.New("smtp timeout must be greater than zero")
	}

	return &smtpSource{
		client:  client,
		from:    from,
		subject: subject,
		timeout: time.Duration(timeoutMs) * time.Millisecond,
	}, nil
}

func (s *smtpSource) Send(
	ctx context.Context,
	target string,
	body string,
) error {
	msg := go_mail.NewMsg()

	if err := msg.From(s.from); err != nil {
		return fmt.Errorf("smtpSource.Send: %w", errors.Join(ErrBuildMessage, err))
	}

	if err := msg.To(target); err != nil {
		return fmt.Errorf("smtpSource.Send: %w", errors.Join(ErrInvalidTarget, err))
	}

	msg.Subject(s.subject)
	msg.SetBodyString(go_mail.TypeTextHTML, body)

	sendCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.client.DialAndSendWithContext(sendCtx, msg); err != nil {
		if strings.Contains(err.Error(), "550 Unauthenticated") {
			return fmt.Errorf("%w: %v", shared.ErrPermanentSMTP, err)
		}
		return fmt.Errorf("smtpSource.Send: %w", errors.Join(shared.ErrTemporarySMTP, err))
	}

	return nil
}
