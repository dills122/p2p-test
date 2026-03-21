package node

import (
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"time"
)

const (
	maxPeerScore         = 100
	minPeerScore         = -100
	peerFailurePenalty   = 5
	peerSuccessReward    = 1
	maxReconnectBackoff  = 30 * time.Second
	baseReconnectBackoff = 500 * time.Millisecond
)

func (node *Node) markPeerHealthy(addr string, rtt time.Duration) {
	now := time.Now()
	node.peers.Update(addr, func(peer *Peer) {
		peer.Status = "online"
		peer.LastSeen = now
		peer.Failures = 0
		peer.CooldownUntil = time.Time{}
		peer.LastRTT = rtt
		peer.Score = minInt(maxPeerScore, peer.Score+peerSuccessReward)
	})
}

func (node *Node) markPeerFailure(addr string) time.Duration {
	now := time.Now()
	var backoff time.Duration
	node.peers.Update(addr, func(peer *Peer) {
		peer.Failures++
		backoff = calculateReconnectBackoff(peer.Failures)
		peer.CooldownUntil = now.Add(backoff)
		peer.Status = "offline"
		peer.Score = maxInt(minPeerScore, peer.Score-peerFailurePenalty)
	})
	return backoff
}

func (node *Node) canAttemptPeer(addr string) (bool, time.Duration) {
	peer, ok := node.peers.Get(addr)
	if !ok || peer.CooldownUntil.IsZero() {
		return true, 0
	}
	remaining := time.Until(peer.CooldownUntil)
	if remaining <= 0 {
		return true, 0
	}
	return false, remaining
}

func (node *Node) AddPeer(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("peer address cannot be empty")
	}
	if addr == node.Addr {
		return fmt.Errorf("cannot add self (%s) as peer", addr)
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return fmt.Errorf("peer address %q is invalid: %w", addr, err)
	}
	if _, exists := node.peers.Get(addr); exists {
		return nil
	}
	if len(node.peers.List()) >= node.maxPeers {
		return fmt.Errorf("peer limit reached (%d), rejecting %s", node.maxPeers, addr)
	}
	node.peers.Add(addr)
	log.Printf("Added peer %s", addr)
	return nil
}

func (node *Node) RemovePeer(addr string) {
	node.peers.Remove(strings.TrimSpace(addr))
}

func (node *Node) ListPeers() []Peer {
	return node.peers.List()
}

func (node *Node) peerAddresses() []string {
	known := node.peers.List()
	seen := make(map[string]struct{})
	addrs := make([]string, 0, len(known)+1)
	add := func(addr string) {
		if addr == "" {
			return
		}
		if _, ok := seen[addr]; ok {
			return
		}
		seen[addr] = struct{}{}
		addrs = append(addrs, addr)
	}
	add(node.Addr)
	for _, peer := range known {
		add(peer.Addr)
	}
	return addrs
}

func calculateReconnectBackoff(failures int) time.Duration {
	if failures <= 0 {
		return 0
	}
	power := math.Pow(2, float64(failures-1))
	backoff := time.Duration(float64(baseReconnectBackoff) * power)
	if backoff > maxReconnectBackoff {
		return maxReconnectBackoff
	}
	return backoff
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
