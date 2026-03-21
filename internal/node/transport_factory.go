package node

import (
	"fmt"
	"strings"
)

const (
	TransportGRPC   = "grpc"
	TransportLibP2P = "libp2p"
)

func resolveTransport(transport string) (Transport, error) {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "", TransportGRPC:
		return NewGRPCTransport(), nil
	case TransportLibP2P:
		return NewLibP2PTransport(), nil
	default:
		return nil, fmt.Errorf("unsupported transport %q (valid: %s, %s)", transport, TransportGRPC, TransportLibP2P)
	}
}

func ResolveTransportForConfig(transport string) (Transport, error) {
	return resolveTransport(transport)
}
