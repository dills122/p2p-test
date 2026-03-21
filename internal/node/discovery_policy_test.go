package node

import "testing"

func TestMergeDiscoveredPeersTrustGate(t *testing.T) {
	n := New(Config{
		NodeName:           "node-1",
		NodeAddr:           "127.0.0.1:21000",
		MaxPeers:           10,
		KnownPeerAddresses: []string{"127.0.0.1:21001"},
	})

	source := "127.0.0.1:21001"
	target := "127.0.0.1:21002"

	// Source is unknown/offline by default, so discovery should be ignored.
	n.mergeDiscoveredPeerAddresses(source, []string{target})
	if _, ok := n.peers.Get(target); ok {
		t.Fatal("expected discovered peer to be ignored from untrusted source")
	}

	n.markPeerHealthy(source, 0)
	n.mergeDiscoveredPeerAddresses(source, []string{target})
	if _, ok := n.peers.Get(target); !ok {
		t.Fatal("expected discovered peer to be accepted from trusted source")
	}
}
