package node

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func (node *Node) markPeerHealthy(addr string) {
	node.peers.UpdateStatus(addr, "online")
	node.peers.UpdateLastSeen(addr, time.Now())
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
