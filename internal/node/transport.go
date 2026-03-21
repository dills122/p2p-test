package node

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	ping "github.com/dills122/p2p-test/pkg/ping"
	"github.com/dills122/p2p-test/pkg/protocol"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type Transport interface {
	Dial(address string) (*grpc.ClientConn, error)
	Ping(ctx context.Context, targetAddr string, selfAddr string, selfPubKey string, envelope protocol.Envelope, message string) (*ping.PingReply, []string, error)
}

func NewGRPCTransport() Transport {
	return &grpcTransport{}
}

type grpcTransport struct{}

func (t *grpcTransport) Dial(address string) (*grpc.ClientConn, error) {
	opts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(100)),
		grpc_retry.WithCodes(codes.NotFound, codes.Aborted),
	}
	conn, err := grpc.Dial(address,
		grpc.WithInsecure(),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(opts...)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to %s: %w", address, err)
	}
	return conn, nil
}

func (t *grpcTransport) Ping(ctx context.Context, targetAddr string, selfAddr string, selfPubKey string, envelope protocol.Envelope, message string) (*ping.PingReply, []string, error) {
	conn, err := t.Dial(targetAddr)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	client := ping.NewPingServiceClient(conn)
	ctx = metadata.AppendToOutgoingContext(ctx, selfMetadataKey, selfAddr)
	if strings.TrimSpace(selfPubKey) != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, selfPublicKeyMetadataKey, selfPubKey)
	}
	if strings.TrimSpace(envelope.ID) != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, messageIDMetadataKey, envelope.ID)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, protocolVersionMetadataKey, envelope.Version)
	ctx = metadata.AppendToOutgoingContext(ctx, messageTypeMetadataKey, string(envelope.Type))
	ctx = metadata.AppendToOutgoingContext(ctx, ttlMetadataKey, strconv.Itoa(envelope.TTL))
	ctx = metadata.AppendToOutgoingContext(ctx, hopCountMetadataKey, strconv.Itoa(envelope.HopCount))
	var header metadata.MD
	reply, err := client.PingNode(ctx, &ping.PingRequest{Message: message}, grpc_retry.WithMax(3), grpc.Header(&header))
	if err != nil {
		return nil, nil, err
	}
	return reply, extractPeerAddresses(header), nil
}

func extractPeerAddresses(md metadata.MD) []string {
	if md == nil {
		return nil
	}
	announcePayload := firstHeaderValue(md, peerAnnouncePayloadKey)
	announceSignature := firstHeaderValue(md, peerAnnounceSignatureKey)
	announcePubKey := firstHeaderValue(md, peerAnnouncePubKeyKey)
	if announcePayload != "" && announceSignature != "" && announcePubKey != "" {
		announce, err := verifySignedPeerAnnounce(announcePayload, announceSignature, announcePubKey)
		if err == nil {
			return announce.Peers
		}
		log.Printf("ignoring untrusted peer announce: %v", err)
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

func firstHeaderValue(md metadata.MD, key string) string {
	values := md.Get(key)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
