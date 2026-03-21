package node

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestBuildAndVerifySignedPeerAnnounce(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	payload, signature, pubKey, err := buildSignedPeerAnnounce("127.0.0.1:10000", []string{
		"127.0.0.1:10001",
		"127.0.0.1:10002",
	}, privateKey)
	if err != nil {
		t.Fatalf("build signed announce: %v", err)
	}

	announce, err := verifySignedPeerAnnounce(payload, signature, pubKey)
	if err != nil {
		t.Fatalf("verify signed announce: %v", err)
	}
	if announce.Origin != "127.0.0.1:10000" {
		t.Fatalf("unexpected origin: %s", announce.Origin)
	}
	if len(announce.Peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(announce.Peers))
	}
}

func TestVerifySignedPeerAnnounceRejectsTamper(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	payload, signature, pubKey, err := buildSignedPeerAnnounce("127.0.0.1:10000", []string{"127.0.0.1:10001"}, privateKey)
	if err != nil {
		t.Fatalf("build signed announce: %v", err)
	}

	// tamper payload while keeping signature same
	tamperedPayload := payload[:len(payload)-2] + "AA"
	if _, err := verifySignedPeerAnnounce(tamperedPayload, signature, pubKey); err == nil {
		t.Fatal("expected tampered payload verification error")
	}
}
