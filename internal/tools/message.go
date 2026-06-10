package tools

import (
	"errors"
	"fmt"
	"strings"
)

const verificationLinkPlaceholder string = "{{verification_link}}"

type messageTool struct {
	template string
}

func NewMessageTool(template string) (*messageTool, error) {
	template = strings.TrimSpace(template)

	if template == "" {
		return nil, errors.New("message template is empty")
	}

	if !strings.Contains(template, verificationLinkPlaceholder) {
		return nil, fmt.Errorf(
			"message template must contain placeholder %s",
			verificationLinkPlaceholder,
		)
	}
	return &messageTool{
		template: template,
	}, nil
}

// BuildMessage заменяет плейсхолдер {{verification_link}} в шаблоне на реальную ссылку для верификации.
func (m *messageTool) BuildMessage(verificationLink string) string {
	return strings.ReplaceAll(
		m.template,
		verificationLinkPlaceholder,
		verificationLink,
	)
}
