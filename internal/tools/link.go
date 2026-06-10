package tools

import (
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

type linkTool struct {
	varificationUrl string
}

func NewLinkTool(varificationUrl string) (*linkTool, error) {
	if _, err := url.Parse(varificationUrl); err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}
	return &linkTool{
		varificationUrl: varificationUrl,
	}, nil
}

// BuildVerificationLink строит ссылку для верификации на основе базового URL, verificationID и hash.
func (l *linkTool) BuildVerificationLink(
	verificationID uuid.UUID,
	hash string,
) string {
	return fmt.Sprintf(
		"%s/email/verify/%s/%s",
		l.varificationUrl,
		verificationID.String(),
		hash,
	)
}
