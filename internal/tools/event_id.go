package tools

import (
	"fmt"

	"github.com/google/uuid"
)

type eventIDTool struct {
	namespace uuid.UUID
}

// В конструкторе задается namespace, при необходимости можно передавать в конструктор
func NewGenerator() *eventIDTool {
	return &eventIDTool{
		namespace: uuid.NameSpaceOID,
	}
}

// BuildDerivedID строит UUID на основе sourceEventID и eventType, используя алгоритм SHA-1.
//
//	Это позволяет гарантировать, что для одного и того же eventID и eventType будет всегда один и тот же derived ID.
func (g *eventIDTool) BuildDerivedID(
	eventID uuid.UUID,
	eventType string,
) uuid.UUID {
	return uuid.NewSHA1(
		g.namespace,
		[]byte(fmt.Sprintf("%s:%s", eventID.String(), eventType)),
	)
}
