package node

import (
	"sort"
	"sync"
	"time"
)

// PeerRegistry tracks known peers and related metadata.
type PeerRegistry interface {
	Add(addr string)
	AddMany(addrs []string)
	Remove(addr string)
	List() []Peer
	UpdateStatus(addr string, status string)
	UpdateLastSeen(addr string, ts time.Time)
}

type memoryPeerRegistry struct {
	mu    sync.RWMutex
	self  string
	peers map[string]Peer
}

func NewPeerRegistry(selfAddr string, bootstrap []string) PeerRegistry {
	reg := &memoryPeerRegistry{
		self:  selfAddr,
		peers: make(map[string]Peer),
	}
	reg.AddMany(bootstrap)
	return reg
}

func (m *memoryPeerRegistry) Add(addr string) {
	if addr == "" || addr == m.self {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.peers[addr]; ok {
		return
	}
	m.peers[addr] = Peer{Addr: addr, Status: "unknown"}
}

func (m *memoryPeerRegistry) AddMany(addrs []string) {
	for _, addr := range addrs {
		m.Add(addr)
	}
}

func (m *memoryPeerRegistry) Remove(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.peers, addr)
}

func (m *memoryPeerRegistry) List() []Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]Peer, 0, len(m.peers))
	for _, peer := range m.peers {
		out = append(out, peer)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Addr < out[j].Addr
	})
	return out
}

func (m *memoryPeerRegistry) UpdateStatus(addr string, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	peer, ok := m.peers[addr]
	if !ok {
		return
	}
	peer.Status = status
	m.peers[addr] = peer
}

func (m *memoryPeerRegistry) UpdateLastSeen(addr string, ts time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	peer, ok := m.peers[addr]
	if !ok {
		return
	}
	peer.LastSeen = ts
	m.peers[addr] = peer
}
