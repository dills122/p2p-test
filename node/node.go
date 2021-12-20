package node

import (
	"context"
	"log"
	"net"
	"time"

	ping "github.com/dills122/p2p-test/pkg/ping"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
)

type Node struct {
	Name string
	Addr string

	Peers []Peer
	ping.UnimplementedPingServiceServer
}

func New(name string, address string) Node {
	staticPeerAddresses := []Peer{{Addr: "127.0.0.1:10000", Status: "unknown"}, {Addr: "127.0.0.1:10001", Status: "unknown"}, {Addr: "127.0.0.1:10002", Status: "unknown"}, {Addr: "127.0.0.1:10003", Status: "unknown"}}
	n := Node{Name: name, Addr: address, Peers: staticPeerAddresses}
	return n
}

func (node *Node) Start() error {
	log.Println("Starting Node")

	go node.startServer()

	log.Println("Started gRPC Server")

	return nil
}

func (node *Node) PingAllNodes(ctx context.Context) {
	log.Println("Executing known peer list")
	for _, peer := range node.Peers {
		reply, err := node.PingNode(ctx, &ping.PingRequest{NodeAddress: peer.Addr})
		if err != nil {
			log.Fatalf("failed to Ping node at Address: %s", peer.Addr)
		}
		log.Printf("Pinged node %s and got a status of %s", peer.Addr, reply.Status)
	}
}

func (node *Node) startServer() {
	log.Println("Starting gRPC Server")
	lis, err := net.Listen("tcp", node.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	ping.RegisterPingServiceServer(grpcServer, node)
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// Nodes Client Ping method
func (node *Node) PingNode(ctx context.Context, stream *ping.PingRequest) (*ping.PingReply, error) {
	client := node.setupClient(stream.NodeAddress)
	pingReply, err := client.PingNode(ctx, stream, grpc_retry.WithMax(3))
	if err != nil {
		log.Fatal("Failed to get status ping")
	}
	return pingReply, err
}

// Sets up a client for a given peer to communicate with
func (node *Node) setupClient(peerAddress string) ping.PingServiceClient {
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
		log.Printf("Unable to connect to %s: %v", peerAddress, err)
	}

	defer conn.Close()

	return ping.NewPingServiceClient(conn)
}
