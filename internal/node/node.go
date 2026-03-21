package node

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"log"
	"time"

	ping "github.com/dills122/p2p-test/pkg/ping"
)

type Node struct {
	Name      string
	Addr      string
	maxPeers  int
	peers     PeerRegistry
	events    *eventBus
	transport Transport
	seen      *messageGuard
	pubKey    ed25519.PublicKey
	privKey   ed25519.PrivateKey
	ping.UnimplementedPingServiceServer
}

const (
	defaultMaxPeers            = 256
	maxDiscoveredPeersPerPing  = 32
	peerMetadataKey            = "peers"
	selfMetadataKey            = "self-addr"
	selfPublicKeyMetadataKey   = "self-pubkey"
	messageIDMetadataKey       = "message-id"
	protocolVersionMetadataKey = "protocol-version"
	messageTypeMetadataKey     = "message-type"
	ttlMetadataKey             = "ttl"
	hopCountMetadataKey        = "hop-count"
	peerAnnouncePayloadKey     = "peer-announce-payload"
	peerAnnounceSignatureKey   = "peer-announce-signature"
	peerAnnouncePubKeyKey      = "peer-announce-pubkey"
)

func New(config Config) *Node {
	bootstrap := sanitizeBootstrap(config.NodeAddr, config.KnownPeerAddresses)
	if len(bootstrap) == 0 {
		bootstrap = sanitizeBootstrap(config.NodeAddr, defaultPeerAddresses)
	}
	transport, err := resolveTransport(config.Transport)
	if err != nil {
		log.Printf("invalid transport %q, defaulting to %s: %v", config.Transport, TransportGRPC, err)
		transport = NewGRPCTransport()
	}

	maxPeers := config.MaxPeers
	if maxPeers <= 0 {
		maxPeers = defaultMaxPeers
	}

	registry := NewPeerRegistry(config.NodeAddr, bootstrap)
	pubKey, privKey, keyErr := ed25519.GenerateKey(rand.Reader)
	if keyErr != nil {
		log.Printf("failed to generate node signing keypair, signed peer announce disabled: %v", keyErr)
	}
	n := &Node{
		Name:      config.NodeName,
		Addr:      config.NodeAddr,
		maxPeers:  maxPeers,
		peers:     registry,
		events:    newEventBus(),
		transport: transport,
		seen:      newMessageGuard(5 * time.Minute),
		pubKey:    pubKey,
		privKey:   privKey,
	}
	return n
}

func (node *Node) Start() {
	log.Println("Starting Node")

	StartServer(node)
}

func (node *Node) Subscribe(fn func(Event)) func() {
	return node.events.Subscribe(fn)
}

func (node *Node) emitEvent(evt Event) {
	node.events.Emit(evt)
}

func (node *Node) publicKeyBase64() string {
	if len(node.pubKey) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(node.pubKey)
}
