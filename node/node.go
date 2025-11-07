package node

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
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

	Peers []Peer
	ping.UnimplementedPingServiceServer
}

var defaultPeerAddresses = []string{
	"127.0.0.1:10000",
	"127.0.0.1:10001",
	"127.0.0.1:10002",
	"127.0.0.1:10003",
}

func New(config Config) Node {
	peers := buildPeers(config.NodeAddr, config.KnownPeerAddresses)
	if len(peers) == 0 {
		peers = buildPeers(config.NodeAddr, defaultPeerAddresses)
	}
	n := Node{Name: config.NodeName, Addr: config.NodeAddr, Peers: peers}
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
	log.Println("Executing known peer list")
	for _, peer := range node.Peers {
		if peer.Addr == node.Addr {
			continue
		}
		reply, err := node.pingPeer(ctx, peer.Addr, msg)
		if err != nil {
			log.Printf("failed to ping node at address %s: %v", peer.Addr, err)
			continue
		}
		log.Printf("Pinged node %s and got a status of %d", peer.Addr, reply.Status)
	}
}

func (node *Node) PingOtherNode(peerAddr *string, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	pingReply, err := node.pingPeer(ctx, *peerAddr, message)
	if err != nil {
		log.Fatalf("Failed to get status ping: %v", err)
	}
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

func buildPeers(selfAddr string, addresses []string) []Peer {
	if len(addresses) == 0 {
		return []Peer{}
	}
	seen := make(map[string]struct{})
	var peers []Peer
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
		peers = append(peers, Peer{Addr: addr, Status: "unknown"})
		seen[addr] = struct{}{}
	}
	return peers
}
