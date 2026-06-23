package fakes

import (
	"context"
	"sync"
)

// PublishedEvent records a single Publish call for assertion.
type PublishedEvent struct {
	EventName string
	Payload   any
}

// EventPublisherFake is a hand-written test double for domain.EventPublisher.
// It captures published events so step definitions can assert on them.
type EventPublisherFake struct {
	mu     sync.Mutex
	events []PublishedEvent
}

// NewEventPublisherFake returns an empty fake.
func NewEventPublisherFake() *EventPublisherFake {
	return &EventPublisherFake{}
}

// Publish records the event.
func (f *EventPublisherFake) Publish(_ context.Context, eventName string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, PublishedEvent{EventName: eventName, Payload: payload})
	return nil
}

// Published returns a snapshot of all captured events.
func (f *EventPublisherFake) Published() []PublishedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]PublishedEvent, len(f.events))
	copy(out, f.events)
	return out
}

// HasEvent returns true if an event with the given name was published.
func (f *EventPublisherFake) HasEvent(name string) bool {
	for _, e := range f.Published() {
		if e.EventName == name {
			return true
		}
	}
	return false
}
