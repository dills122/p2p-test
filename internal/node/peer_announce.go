package node

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type peerAnnounce struct {
	Origin    string   `json:"origin"`
	Timestamp int64    `json:"timestamp"`
	Peers     []string `json:"peers"`
}

func buildSignedPeerAnnounce(origin string, peers []string, privateKey ed25519.PrivateKey) (string, string, string, error) {
	if len(privateKey) == 0 {
		return "", "", "", fmt.Errorf("private key is not set")
	}

	normalized := normalizePeerList(peers)
	payload := peerAnnounce{
		Origin:    strings.TrimSpace(origin),
		Timestamp: time.Now().UTC().Unix(),
		Peers:     normalized,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", "", err
	}

	signature := ed25519.Sign(privateKey, payloadBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return base64.StdEncoding.EncodeToString(payloadBytes),
		base64.StdEncoding.EncodeToString(signature),
		base64.StdEncoding.EncodeToString(publicKey),
		nil
}

func verifySignedPeerAnnounce(payloadB64 string, signatureB64 string, pubKeyB64 string) (peerAnnounce, error) {
	var announce peerAnnounce

	payloadBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payloadB64))
	if err != nil {
		return announce, fmt.Errorf("decode announce payload: %w", err)
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureB64))
	if err != nil {
		return announce, fmt.Errorf("decode announce signature: %w", err)
	}
	publicKeyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(pubKeyB64))
	if err != nil {
		return announce, fmt.Errorf("decode announce public key: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKeyBytes), payloadBytes, signatureBytes) {
		return announce, fmt.Errorf("invalid peer announce signature")
	}
	if err := json.Unmarshal(payloadBytes, &announce); err != nil {
		return announce, fmt.Errorf("unmarshal announce payload: %w", err)
	}
	announce.Origin = strings.TrimSpace(announce.Origin)
	announce.Peers = normalizePeerList(announce.Peers)
	return announce, nil
}

func normalizePeerList(peers []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(peers))
	for _, peer := range peers {
		peer = strings.TrimSpace(peer)
		if peer == "" {
			continue
		}
		if _, ok := seen[peer]; ok {
			continue
		}
		seen[peer] = struct{}{}
		out = append(out, peer)
	}
	sort.Strings(out)
	return out
}
