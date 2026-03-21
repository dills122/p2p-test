package node

import (
	"context"
	"fmt"

	ping "github.com/dills122/p2p-test/pkg/ping"
	"github.com/dills122/p2p-test/pkg/protocol"
	"google.golang.org/grpc"
)

type libp2pTransport struct{}

func NewLibP2PTransport() Transport {
	return &libp2pTransport{}
}

func (t *libp2pTransport) Dial(address string) (*grpc.ClientConn, error) {
	return nil, fmt.Errorf("libp2p transport Dial is not implemented yet (address=%s)", address)
}

func (t *libp2pTransport) Ping(ctx context.Context, targetAddr string, selfAddr string, selfPubKey string, envelope protocol.Envelope, message string) (*ping.PingReply, []string, error) {
	_ = ctx
	_ = selfAddr
	_ = selfPubKey
	_ = envelope
	_ = message
	return nil, nil, fmt.Errorf("libp2p transport Ping is not implemented yet (target=%s)", targetAddr)
}
