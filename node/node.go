package node

import (
	"context"
	"fmt"
	"log"
	"time"

	ping "github.com/dills122/p2p-test/pkg/ping"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

func (node *Node) Start() {
	log.Println("Starting Node")

	StartServer(node.Addr)
}

func (node *Node) PingAllNodes(ctx context.Context) {
	log.Println("Executing known peer list")
	for _, peer := range node.Peers {
		reply, err := node.PingNode(ctx, &ping.PingRequest{Message: "Pinging"}) // TODO update with a pass through for message
		if err != nil {
			log.Fatalf("failed to Ping node at Address: %s", peer.Addr)
		}
		log.Printf("Pinged node %s and got a status of %d", peer.Addr, reply.Status)
	}
}

func (node *Node) PingOtherNode(peerAddr *string, message string) {
	client, conn := node.setupClient(*peerAddr)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	pingReply, err := client.PingNode(ctx, &ping.PingRequest{Message: message}, grpc_retry.WithMax(3))
	defer conn.Close()
	defer cancel()
	if err != nil {
		log.Fatalf("Failed to get status ping: %v", err)
	}
	fmt.Printf("Reply received from node %s with status: %d and message: %s \n", *peerAddr, pingReply.Status, pingReply.Message)
}

// ***************
// PRIVATE METHODS
// ***************

func (node *Node) setupClient(peerAddress string) (ping.PingServiceClient, grpc.ClientConn) {
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
		log.Fatalf("Unable to connect to %s: %v", peerAddress, err)
	}

	return ping.NewPingServiceClient(conn), *conn
}
