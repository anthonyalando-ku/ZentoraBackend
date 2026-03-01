// internal/websocket/handler.go
package websocket

import (
	"context"
	wstypes "diary-service/internal/domain/websocket"
)

// MessageHandler interface that each module must implement
type MessageHandler interface {
	// HandleMessage processes messages for this handler's domain
	HandleMessage(ctx context.Context, client *Client, msg *wstypes.WSMessage) error
	
	// SupportedEvents returns the list of event types this handler supports
	SupportedEvents() []wstypes.EventType
}

// HandlerRegistry manages all message handlers
type HandlerRegistry struct {
	handlers map[wstypes.EventType]MessageHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[wstypes.EventType]MessageHandler),
	}
}

// Register registers a handler for its supported events
func (r *HandlerRegistry) Register(handler MessageHandler) {
	for _, eventType := range handler.SupportedEvents() {
		r.handlers[eventType] = handler
	}
}

// GetHandler returns the handler for a given event type
func (r *HandlerRegistry) GetHandler(eventType wstypes.EventType) (MessageHandler, bool) {
	handler, exists := r.handlers[eventType]
	return handler, exists
}