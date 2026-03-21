package node

import "testing"

func TestAddPeerRespectsMaxPeers(t *testing.T) {
	n := New(Config{
		NodeName:           "node-1",
		NodeAddr:           "127.0.0.1:10000",
		MaxPeers:           2,
		KnownPeerAddresses: []string{"127.0.0.1:11001"},
	})

	if err := n.AddPeer("127.0.0.1:11002"); err != nil {
		t.Fatalf("unexpected add peer error: %v", err)
	}
	if err := n.AddPeer("127.0.0.1:11003"); err == nil {
		t.Fatal("expected max peer limit error")
	}
}
