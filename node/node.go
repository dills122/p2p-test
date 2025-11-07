package node

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	ping "github.com/dills122/p2p-test/pkg/ping"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

type Node struct {
	Name string
	Addr string

	peers          PeerRegistry
	listenersMu    sync.RWMutex
	listeners      map[int]func(Event)
	nextListenerID int
	ping.UnimplementedPingServiceServer
}

var defaultPeerAddresses = []string{
	"127.0.0.1:10000",
	"127.0.0.1:10001",
	"127.0.0.1:10002",
	"127.0.0.1:10003",
}

const (
	peerMetadataKey = "peers"
	selfMetadataKey = "self-addr"
	EventTypeSent   = "sent"
	EventTypeRecv   = "received"
	EventTypeError  = "error"
	EventTypeInfo   = "info"
)

type Event struct {
	Type      string
	Peer      string
	Message   string
	Err       error
	Timestamp time.Time
}

func New(config Config) *Node {
	bootstrap := sanitizeBootstrap(config.NodeAddr, config.KnownPeerAddresses)
	if len(bootstrap) == 0 {
		bootstrap = sanitizeBootstrap(config.NodeAddr, defaultPeerAddresses)
	}
	registry := NewPeerRegistry(config.NodeAddr, bootstrap)
	n := &Node{Name: config.NodeName, Addr: config.NodeAddr, peers: registry}
	return n
}

func (node *Node) Start() {
	log.Println("Starting Node")

	StartServer(node)
}

func (node *Node) PingAllNodes(ctx context.Context, msg string) {
	if len(msg) <= 0 {
		msg = "Pinging"
	}
	knownPeers := node.peers.List()
	if len(knownPeers) == 0 {
		log.Println("No known peers to ping")
		return
	}
	log.Println("Executing known peer list")

	visited := make(map[string]struct{})
	queue := make([]string, 0, len(knownPeers))
	for _, peer := range knownPeers {
		queue = append(queue, peer.Addr)
	}

	for len(queue) > 0 {
		peerAddr := queue[0]
		queue = queue[1:]
		if peerAddr == "" || peerAddr == node.Addr {
			continue
		}
		if _, ok := visited[peerAddr]; ok {
			continue
		}
		visited[peerAddr] = struct{}{}

		peerCtx, cancel := context.WithTimeout(ctx, time.Second*3)
		reply, discovered, err := node.pingPeer(peerCtx, peerAddr, msg)
		cancel()
		if err != nil {
			log.Printf("failed to ping node at address %s: %v", peerAddr, err)
			node.emitEvent(Event{
				Type:      EventTypeError,
				Peer:      peerAddr,
				Message:   msg,
				Err:       err,
				Timestamp: time.Now(),
			})
			continue
		}
		node.markPeerHealthy(peerAddr)
		node.mergePeerAddresses(discovered)
		queue = append(queue, discovered...)
		node.emitEvent(Event{
			Type:      EventTypeSent,
			Peer:      peerAddr,
			Message:   msg,
			Timestamp: time.Now(),
		})
		log.Printf("Pinged node %s and got a status of %d", peerAddr, reply.Status)
	}
}

func (node *Node) PingOtherNode(peerAddr *string, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	pingReply, discovered, err := node.pingPeer(ctx, *peerAddr, message)
	if err != nil {
		log.Fatalf("Failed to get status ping: %v", err)
	}
	node.markPeerHealthy(*peerAddr)
	node.mergePeerAddresses(discovered)
	node.emitEvent(Event{
		Type:      EventTypeSent,
		Peer:      *peerAddr,
		Message:   message,
		Timestamp: time.Now(),
	})
	fmt.Printf("Reply received from node %s with status: %d and message: %s \n", *peerAddr, pingReply.Status, pingReply.Message)
}

func (node *Node) CheckIfReady() bool {
	conn, err := node.setupClient(node.Addr)
	if err != nil {
		log.Fatalf("Unable to connect to health service on %s: %v", node.Addr, err)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
	ctx := context.Background()
	stream, err := client.Watch(ctx, &grpc_health_v1.HealthCheckRequest{})

	if err != nil {
		log.Fatalf("open stream error %v", err)
	}

	done := make(chan bool)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- false //means stream is finished
				return
			}
			if err != nil {
				log.Fatalf("cannot receive %v", err)
			}
			log.Printf("Resp received: %s", resp.Status)
			if resp.Status == 1 { //SERVING
				done <- true //means stream is finished
				return
			}
		}
	}()

	isAvailable := <-done
	return isAvailable
}

// ***************
// PRIVATE METHODS
// ***************

func (node *Node) setupClient(peerAddress string) (*grpc.ClientConn, error) {
	log.Printf("Creating client for node %s", peerAddress)
	opts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(100 * time.Millisecond)),
		grpc_retry.WithCodes(codes.NotFound, codes.Aborted),
	}
	conn, err := grpc.Dial(peerAddress,
		grpc.WithInsecure(),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(opts...)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to %s: %w", peerAddress, err)
	}

	return conn, nil
}

func (node *Node) pingPeer(ctx context.Context, peerAddr string, message string) (*ping.PingReply, []string, error) {
	conn, err := node.setupClient(peerAddr)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	client := ping.NewPingServiceClient(conn)
	ctx = metadata.AppendToOutgoingContext(ctx, selfMetadataKey, node.Addr)
	var header metadata.MD
	reply, err := client.PingNode(ctx, &ping.PingRequest{Message: message}, grpc_retry.WithMax(3), grpc.Header(&header))
	if err != nil {
		return nil, nil, err
	}
	discovered := extractPeerAddresses(header)
	return reply, discovered, nil
}

func sanitizeBootstrap(selfAddr string, addresses []string) []string {
	if len(addresses) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{})
	var peers []string
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr == "" || addr == selfAddr {
			continue
		}
		if _, _, err := net.SplitHostPort(addr); err != nil {
			log.Printf("Skipping peer address %q: %v", addr, err)
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		peers = append(peers, addr)
		seen[addr] = struct{}{}
	}
	return peers
}

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

func (node *Node) mergePeerAddresses(addresses []string) {
	for _, addr := range addresses {
		if err := node.AddPeer(addr); err != nil {
			continue
		}
	}
}

func extractPeerAddresses(md metadata.MD) []string {
	if md == nil {
		return nil
	}
	values := md.Get(peerMetadataKey)
	var addresses []string
	for _, value := range values {
		for _, addr := range strings.Split(value, ",") {
			addr = strings.TrimSpace(addr)
			if addr == "" {
				continue
			}
			addresses = append(addresses, addr)
		}
	}
	return addresses
}

func (node *Node) Subscribe(fn func(Event)) func() {
	node.listenersMu.Lock()
	defer node.listenersMu.Unlock()
	if node.listeners == nil {
		node.listeners = make(map[int]func(Event))
	}
	id := node.nextListenerID
	node.nextListenerID++
	node.listeners[id] = fn
	return func() {
		node.listenersMu.Lock()
		defer node.listenersMu.Unlock()
		delete(node.listeners, id)
	}
}

func (node *Node) emitEvent(evt Event) {
	node.listenersMu.RLock()
	defer node.listenersMu.RUnlock()
	for _, listener := range node.listeners {
		go listener(evt)
	}
}
