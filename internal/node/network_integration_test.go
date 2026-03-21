package node

import (
	"strings"
	"testing"
	"time"
)

func TestNodeRestartRejoinE2E(t *testing.T) {
	addrA := freeAddress(t)
	addrB := freeAddress(t)

	nodeA := New(Config{
		NodeName:           "node-a",
		NodeAddr:           addrA,
		KnownPeerAddresses: []string{addrB},
	})
	nodeB := New(Config{
		NodeName:           "node-b",
		NodeAddr:           addrB,
		KnownPeerAddresses: []string{addrA},
	})

	stopA := startExistingNodeServer(t, nodeA)
	defer stopA()
	stopB := startExistingNodeServer(t, nodeB)

	if err := nodeA.PingOtherNode(addrB, "initial-join"); err != nil {
		stopB()
		t.Fatalf("initial ping failed: %v", err)
	}

	stopB()

	if err := nodeA.PingOtherNode(addrB, "during-restart"); err == nil {
		t.Fatal("expected ping to fail while node-b is stopped")
	}

	nodeB2 := New(Config{
		NodeName:           "node-b-restarted",
		NodeAddr:           addrB,
		KnownPeerAddresses: []string{addrA},
	})
	stopB2 := startExistingNodeServer(t, nodeB2)
	defer stopB2()

	waitForNoError(t, 5*time.Second, 150*time.Millisecond, func() error {
		return nodeA.PingOtherNode(addrB, "after-restart")
	})

	peerB := findPeerByAddr(t, nodeA.ListPeers(), addrB)
	if peerB.Status != "online" {
		t.Fatalf("expected node-b status online after rejoin, got %q", peerB.Status)
	}
}

func TestTemporaryPartitionHealingE2E(t *testing.T) {
	addrA := freeAddress(t)
	addrB := freeAddress(t)
	addrC := freeAddress(t)

	nodeA := New(Config{
		NodeName:           "node-a",
		NodeAddr:           addrA,
		KnownPeerAddresses: []string{addrB},
	})
	nodeB := New(Config{
		NodeName:           "node-b",
		NodeAddr:           addrB,
		KnownPeerAddresses: []string{addrA, addrC},
	})
	nodeC := New(Config{
		NodeName:           "node-c",
		NodeAddr:           addrC,
		KnownPeerAddresses: []string{addrB},
	})

	stopA := startExistingNodeServer(t, nodeA)
	defer stopA()
	stopB := startExistingNodeServer(t, nodeB)
	stopC := startExistingNodeServer(t, nodeC)
	defer stopC()

	if err := nodeC.PingOtherNode(addrB, "seed-c-to-b"); err != nil {
		stopB()
		t.Fatalf("seed c->b failed: %v", err)
	}
	if err := nodeA.PingOtherNode(addrB, "seed-a-to-b"); err != nil {
		stopB()
		t.Fatalf("seed a->b failed: %v", err)
	}

	stopB()

	if err := nodeA.PingOtherNode(addrB, "partition-a"); err == nil {
		t.Fatal("expected node-a ping to fail during partition")
	}
	if err := nodeC.PingOtherNode(addrB, "partition-c"); err == nil {
		t.Fatal("expected node-c ping to fail during partition")
	}

	nodeB2 := New(Config{
		NodeName:           "node-b-restarted",
		NodeAddr:           addrB,
		KnownPeerAddresses: []string{addrA, addrC},
	})
	stopB2 := startExistingNodeServer(t, nodeB2)
	defer stopB2()

	waitForNoError(t, 5*time.Second, 150*time.Millisecond, func() error {
		return nodeA.PingOtherNode(addrB, "heal-a")
	})
	waitForNoError(t, 5*time.Second, 150*time.Millisecond, func() error {
		return nodeC.PingOtherNode(addrB, "heal-c")
	})

	peerBFromA := findPeerByAddr(t, nodeA.ListPeers(), addrB)
	if peerBFromA.Status != "online" {
		t.Fatalf("expected node-b online from node-a view, got %q", peerBFromA.Status)
	}
	peerAFromC := findPeerByAddr(t, nodeC.ListPeers(), addrA)
	if peerAFromC.Addr == "" {
		t.Fatal("expected node-c to re-learn node-a after partition healing")
	}
}

func waitForNoError(t *testing.T, timeout time.Duration, interval time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := fn(); err == nil {
			return
		} else {
			lastErr = err
		}
		time.Sleep(interval)
	}
	if lastErr == nil {
		t.Fatalf("operation did not succeed before timeout (%s)", timeout)
	}
	t.Fatalf("operation did not succeed before timeout (%s): %v", timeout, lastErr)
}

func findPeerByAddr(t *testing.T, peers []Peer, addr string) Peer {
	t.Helper()
	for _, peer := range peers {
		if strings.TrimSpace(peer.Addr) == strings.TrimSpace(addr) {
			return peer
		}
	}
	t.Fatalf("peer %s not found in peer list", addr)
	return Peer{}
}
