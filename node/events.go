package node

import (
	"sync"
	"time"
)

type EventType string

const (
	EventTypeSent  EventType = "sent"
	EventTypeRecv  EventType = "received"
	EventTypeError EventType = "error"
	EventTypeInfo  EventType = "info"
)

type Event struct {
	Type      EventType
	Peer      string
	Message   string
	Err       error
	Timestamp time.Time
}

type eventBus struct {
	mu        sync.RWMutex
	subs      map[int]func(Event)
	nextSubID int
}

func newEventBus() *eventBus {
	return &eventBus{subs: make(map[int]func(Event))}
}

func (b *eventBus) Subscribe(fn func(Event)) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextSubID
	b.nextSubID++
	b.subs[id] = fn
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subs, id)
	}
}

func (b *eventBus) Emit(evt Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, fn := range b.subs {
		go fn(evt)
	}
}
