package node

import (
	"context"
	"log"

	ping "github.com/dills122/p2p-test/node/out"
)

type Node struct {
	Name string
	Addr string

	Peers map[string]ping.PingServiceClient
}

func (node *Node) PingNode(ctx context.Context, stream *ping.PingRequest) (*ping.PingReply, error) {
	client := node.Peers[stream.NodeAddress]
	pingReply, err := client.PingNode(ctx, stream)
	if err != nil {
		log.Fatal("Failed to get status ping")
	}
	return pingReply, err
}
