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
)

type Node struct {
	Name string
	Addr string

	peers PeerRegistry
	ping.UnimplementedPingServiceServer
}

var defaultPeerAddresses = []string{
	"127.0.0.1:10000",
	"127.0.0.1:10001",
	"127.0.0.1:10002",
	"127.0.0.1:10003",
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

	StartServer(node.Addr)
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
	var wg sync.WaitGroup
	for _, peer := range knownPeers {
		peerAddr := peer.Addr
		wg.Add(1)
		go func() {
			defer wg.Done()
			peerCtx, cancel := context.WithTimeout(ctx, time.Second*3)
			defer cancel()
			reply, err := node.pingPeer(peerCtx, peerAddr, msg)
			if err != nil {
				log.Printf("failed to ping node at address %s: %v", peerAddr, err)
				return
			}
			node.markPeerHealthy(peerAddr)
			log.Printf("Pinged node %s and got a status of %d", peerAddr, reply.Status)
		}()
	}
	wg.Wait()
}

func (node *Node) PingOtherNode(peerAddr *string, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	pingReply, err := node.pingPeer(ctx, *peerAddr, message)
	if err != nil {
		log.Fatalf("Failed to get status ping: %v", err)
	}
	node.markPeerHealthy(*peerAddr)
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

func (node *Node) pingPeer(ctx context.Context, peerAddr string, message string) (*ping.PingReply, error) {
	conn, err := node.setupClient(peerAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := ping.NewPingServiceClient(conn)
	return client.PingNode(ctx, &ping.PingRequest{Message: message}, grpc_retry.WithMax(3))
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
