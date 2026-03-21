package node

import (
	"testing"
	"time"
)

func TestCalculateReconnectBackoff(t *testing.T) {
	tests := []struct {
		failures int
		want     time.Duration
	}{
		{failures: 0, want: 0},
		{failures: 1, want: 500 * time.Millisecond},
		{failures: 2, want: 1 * time.Second},
		{failures: 3, want: 2 * time.Second},
		{failures: 7, want: 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(time.Duration(tt.failures).String(), func(t *testing.T) {
			got := calculateReconnectBackoff(tt.failures)
			if got != tt.want {
				t.Fatalf("calculateReconnectBackoff(%d) = %s, want %s", tt.failures, got, tt.want)
			}
		})
	}
}

func TestMarkPeerFailureAndCooldown(t *testing.T) {
	n := New(Config{
		NodeName: "node-1",
		NodeAddr: "127.0.0.1:10000",
	})
	addr := "127.0.0.1:10001"
	if err := n.AddPeer(addr); err != nil {
		t.Fatalf("add peer: %v", err)
	}

	backoff := n.markPeerFailure(addr)
	if backoff <= 0 {
		t.Fatalf("expected positive backoff, got %s", backoff)
	}

	peer, ok := n.peers.Get(addr)
	if !ok {
		t.Fatalf("peer not found")
	}
	if peer.Failures != 1 {
		t.Fatalf("expected 1 failure, got %d", peer.Failures)
	}
	if peer.Status != "offline" {
		t.Fatalf("expected offline status, got %s", peer.Status)
	}
	if peer.CooldownUntil.IsZero() {
		t.Fatal("expected cooldown to be set")
	}
	if allowed, _ := n.canAttemptPeer(addr); allowed {
		t.Fatal("expected peer to be in cooldown")
	}

	n.peers.Update(addr, func(p *Peer) {
		p.CooldownUntil = time.Now().Add(-time.Millisecond)
	})
	if allowed, _ := n.canAttemptPeer(addr); !allowed {
		t.Fatal("expected peer to be available after cooldown expiry")
	}
}
