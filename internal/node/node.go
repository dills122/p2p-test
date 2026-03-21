package node

import (
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
	ping.UnimplementedPingServiceServer
}

const (
	defaultMaxPeers            = 256
	peerMetadataKey            = "peers"
	selfMetadataKey            = "self-addr"
	messageIDMetadataKey       = "message-id"
	protocolVersionMetadataKey = "protocol-version"
	messageTypeMetadataKey     = "message-type"
	ttlMetadataKey             = "ttl"
	hopCountMetadataKey        = "hop-count"
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
	n := &Node{
		Name:      config.NodeName,
		Addr:      config.NodeAddr,
		maxPeers:  maxPeers,
		peers:     registry,
		events:    newEventBus(),
		transport: transport,
		seen:      newMessageGuard(5 * time.Minute),
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
