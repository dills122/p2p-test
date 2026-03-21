package node

import (
	"strings"
	"sync"
	"time"
)

type messageGuard struct {
	mu     sync.Mutex
	seen   map[string]time.Time
	window time.Duration
}

func newMessageGuard(window time.Duration) *messageGuard {
	return &messageGuard{
		seen:   make(map[string]time.Time),
		window: window,
	}
}

func (g *messageGuard) IsDuplicate(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}

	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.window > 0 {
		cutoff := now.Add(-g.window)
		for key, ts := range g.seen {
			if ts.Before(cutoff) {
				delete(g.seen, key)
			}
		}
	}

	if _, ok := g.seen[id]; ok {
		return true
	}
	g.seen[id] = now
	return false
}
