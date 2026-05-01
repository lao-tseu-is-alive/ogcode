package bus

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

type Bus struct {
	mu       sync.RWMutex
	subs     []chan Event
	closed   bool
	bufSize  int
}

func New(bufSize int) *Bus {
	if bufSize <= 0 {
		bufSize = 1024
	}
	return &Bus{bufSize: bufSize}
}

func (b *Bus) Publish(eventType string, properties any) {
	data, err := json.Marshal(properties)
	if err != nil {
		return
	}
	evt := Event{Type: eventType, Properties: data}
	b.mu.RLock()
	for _, ch := range b.subs {
		select {
		case ch <- evt:
		default:
		}
	}
	b.mu.RUnlock()
}

func (b *Bus) SubscribeAll() <-chan Event {
	ch := make(chan Event, b.bufSize)
	b.mu.Lock()
	b.subs = append(b.subs, ch)
	b.mu.Unlock()
	return ch
}

func (b *Bus) Unsubscribe(ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, s := range b.subs {
		if s == ch {
			b.subs = append(b.subs[:i], b.subs[i+1:]...)
			close(s)
			return
		}
	}
}