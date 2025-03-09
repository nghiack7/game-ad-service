package events

import (
	"context"
	"time"
)

type Event struct {
	ID        string    `json:"id"`
	EventType string    `json:"event_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EventHandler func(ctx context.Context, event Event) error

type EventBus interface {
	Publish(ctx context.Context, event Event) error
}

type EventBusImpl struct {
	handlers map[string][]EventHandler
}

func NewEventBus() *EventBusImpl {
	return &EventBusImpl{handlers: make(map[string][]EventHandler)}
}

func (e *EventBusImpl) Publish(ctx context.Context, event Event) error {
	handlers := e.handlers[event.EventType]
	for _, handler := range handlers {
		handler(ctx, event)
	}
	return nil
}

func (e *EventBusImpl) Subscribe(eventType string, handler EventHandler) {
	e.handlers[eventType] = append(e.handlers[eventType], handler)
}
