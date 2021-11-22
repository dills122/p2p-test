package node

import (
	"context"

	ping "github.com/dills122/p2p-test/node/out/ping"
)

type Node struct {
	Name string
	Addr string

	Peers map[string]ping.PingServiceClient
}

func (node *Node) PingNode(ctx context.Context, stream *ping.PingRequest) (*ping.PingReply, error) {
	return &ping.PingReply{nodeAddress: node.Addr, status: "good"}, nil
}
