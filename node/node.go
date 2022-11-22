package node

import (
	"context"
	"fmt"
	"io"
	"log"
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

func New(name string, address string) Node {
	staticPeerAddresses := []Peer{{Addr: "127.0.0.1:10000", Status: "unknown"}, {Addr: "127.0.0.1:10001", Status: "unknown"}, {Addr: "127.0.0.1:10002", Status: "unknown"}, {Addr: "127.0.0.1:10003", Status: "unknown"}}
	n := Node{Name: name, Addr: address, Peers: staticPeerAddresses}
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
		reply, err := node.PingNode(ctx, &ping.PingRequest{Message: msg}) // TODO update with a pass through for message
		if err != nil {
			log.Fatalf("failed to Ping node at Address: %s", peer.Addr)
		}
		log.Printf("Pinged node %s and got a status of %d", peer.Addr, reply.Status)
	}
}

func (node *Node) PingOtherNode(peerAddr *string, message string) {
	conn := node.setupClient(*peerAddr)
	client := ping.NewPingServiceClient(&conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	pingReply, err := client.PingNode(ctx, &ping.PingRequest{Message: message}, grpc_retry.WithMax(3))
	defer conn.Close()
	defer cancel()
	if err != nil {
		log.Fatalf("Failed to get status ping: %v", err)
	}
	fmt.Printf("Reply received from node %s with status: %d and message: %s \n", *peerAddr, pingReply.Status, pingReply.Message)
}

func (node *Node) CheckIfReady() bool {
	conn := node.setupClient(node.Addr)
	client := grpc_health_v1.NewHealthClient(&conn)
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

func (node *Node) setupClient(peerAddress string) grpc.ClientConn {
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

	return *conn
}
