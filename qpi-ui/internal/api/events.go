package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// EventHandler processes a single inbound event for the QPU of the driver that
// sent it. The QPU is implied by the driver, so a handler always acts on the
// calling driver's QPU (RFC 0001 §4). Returning an error means the event is
// logged and dropped: there is no application-level ACK/NACK.
type EventHandler func(ctx context.Context, app core.App, qpuID string, event *Event) error

// EventRegistry is the static map of event types to their server-side handlers.
// The set of types is fixed per QPI-UI version and known at compile time, so the
// framework grows by registering a handler for each new type (RFC 0001 §7).
type EventRegistry struct {
	handlers map[EventType]EventHandler
}

// NewEventRegistry returns an empty registry.
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{handlers: make(map[EventType]EventHandler)}
}

// Register binds a handler to an event type, replacing any existing one.
func (r *EventRegistry) Register(eventType EventType, handler EventHandler) {
	r.handlers[eventType] = handler
}

// Handles reports whether a handler is registered for the event type.
func (r *EventRegistry) Handles(eventType EventType) bool {
	_, ok := r.handlers[eventType]
	return ok
}

// Dispatch routes an inbound event to its registered handler. An event whose
// type has no handler, or whose handler returns an error, is logged and dropped
// (RFC 0001 §4). The handler error is returned so callers can observe it; the
// transport loop ignores it and keeps listening.
func (r *EventRegistry) Dispatch(ctx context.Context, app core.App, qpuID string, event *Event) error {
	handler, ok := r.handlers[event.Type]
	if !ok {
		log.Printf("[Events %s] dropping event %s: no handler for type %q", qpuID, event.ID, event.Type)
		return nil
	}

	if err := handler(ctx, app, qpuID, event); err != nil {
		log.Printf("[Events %s] dropping event %s of type %q: %v", qpuID, event.ID, event.Type, err)
		return err
	}

	return nil
}

// NewEvent builds an outbound event of the given type from driver, marshalling
// payload into the envelope and assigning a fresh id and timestamp.
func NewEvent(driver string, eventType EventType, payload any) (*Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal %s payload: %w", eventType, err)
	}

	event := &Event{
		Driver:  driver,
		Type:    eventType,
		Payload: raw,
	}
	event.SetDefaults()
	return event, nil
}

// generateEventID creates a random identifier for an event envelope, mirroring
// the token generator's use of crypto/rand with a timestamp fallback.
func generateEventID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(time.Now().UnixNano() % 256)
			time.Sleep(1 * time.Nanosecond)
		}
	}
	return "evt_" + hex.EncodeToString(b)
}
