package node

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	ping "github.com/dills122/p2p-test/pkg/ping"
	"github.com/dills122/p2p-test/pkg/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
)

const (
	OFFLINE int = 0
	READY   int = 1
	CLOSED  int = 2
)

type Service struct {
	node *Node
	ping.UnimplementedPingServiceServer
	grpc_health_v1.UnimplementedHealthServer
}

func (service *Service) PingNode(ctx context.Context, stream *ping.PingRequest) (*ping.PingReply, error) {
	log.Printf("Received ping message: %s", stream.Message)
	remote, envelope := service.trackCaller(ctx, stream.Message)
	if service.node != nil {
		if err := service.node.acceptIncomingEnvelope(envelope); err != nil {
			log.Printf("Dropping ping from %s (id=%s): %v", remote, envelope.ID, err)
			return &ping.PingReply{Message: stream.Message, Status: int32(OFFLINE)}, nil
		}
	}
	service.sendPeerMetadata(ctx)
	if service.node != nil {
		service.node.emitEvent(Event{
			Type:      EventTypeRecv,
			MessageID: envelope.ID,
			Peer:      remote,
			Message:   stream.Message,
			Timestamp: time.Now(),
		})
	}
	return &ping.PingReply{Message: stream.Message, Status: int32(READY)}, nil
}

func StartServer(node *Node) {
	addr := node.Addr
	log.Printf("Starting gRPC Server on %s", addr)
	builder := GrpcServerBuilder{}
	s := builder.Build(func(grpcSrv *grpc.Server) {
		setupServerServices(grpcSrv, node)
	})
	err := s.Start(addr)
	if err != nil {
		log.Fatalf("Failed to start gRPC server on %s: %v", addr, err)
	}
	s.AwaitTermination(func() {
		log.Println("Shutting down the server")
	})
}

func setupServerServices(server *grpc.Server, node *Node) {
	ping.RegisterPingServiceServer(server, &Service{node: node})
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())
}

type GrpcServer interface {
	Start(address string) error
	AwaitTermination(shutdownHook func())
	GetListener() net.Listener
}

type GrpcServerBuilder struct {
	options []grpc.ServerOption
}

type grpcServer struct {
	server   *grpc.Server
	listener net.Listener
}

func (s grpcServer) GetListener() net.Listener {
	return s.listener
}

func (sb *GrpcServerBuilder) Build(register func(*grpc.Server)) GrpcServer {
	srv := grpc.NewServer(sb.options...)
	if register != nil {
		register(srv)
	}
	reflection.Register(srv)
	return &grpcServer{srv, nil}
}

// Start the GRPC server
func (s *grpcServer) Start(addr string) error {
	var err error
	s.listener, err = net.Listen("tcp", addr)

	if err != nil {
		msg := fmt.Sprintf("Failed to listen: %v", err)
		return errors.New(msg)
	}

	go s.serv()

	log.Printf("gRPC Server started on %s \n", addr)
	return nil
}

// AwaitTermination makes the program wait for the signal termination
func (s *grpcServer) AwaitTermination(shutdownHook func()) {
	interruptSignal := make(chan os.Signal, 1)
	signal.Notify(interruptSignal, syscall.SIGINT, syscall.SIGTERM)
	<-interruptSignal
	s.cleanup()
	if shutdownHook != nil {
		shutdownHook()
	}
}

func (s *grpcServer) cleanup() {
	log.Println("Stopping the server")
	s.server.GracefulStop()
	log.Println("Closing the listener")
	s.listener.Close()
	log.Println("End of Program")
}

func (s *grpcServer) serv() {
	if err := s.server.Serve(s.listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (service *Service) trackCaller(ctx context.Context, message string) (string, protocol.Envelope) {
	env := protocol.NewEnvelope(protocol.MessageTypePing, "unknown", []byte(message), protocol.DefaultTTL)
	if service.node == nil {
		return "", env
	}
	var remote string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		setEnvelopeMetadata(&env, md)
		addresses := md.Get(selfMetadataKey)
		for _, addr := range addresses {
			addr = strings.TrimSpace(addr)
			if addr == "" {
				continue
			}
			if err := service.node.AddPeer(addr); err != nil {
				continue
			}
			service.node.markPeerHealthy(addr)
			if remote == "" {
				remote = addr
			}
		}
	}
	if remote == "" {
		if pr, ok := peer.FromContext(ctx); ok && pr.Addr != nil {
			remote = pr.Addr.String()
			if err := service.node.AddPeer(remote); err == nil {
				service.node.markPeerHealthy(remote)
			}
		}
	}
	if remote != "" {
		env.Origin = remote
	}
	return remote, env
}

func (service *Service) sendPeerMetadata(ctx context.Context) {
	if service.node == nil {
		return
	}
	peers := service.node.peerAddresses()
	if len(peers) == 0 {
		return
	}
	value := strings.Join(peers, ",")
	if err := grpc.SetHeader(ctx, metadata.Pairs(peerMetadataKey, value)); err != nil {
		log.Printf("failed to send peer metadata: %v", err)
	}
}

func setEnvelopeMetadata(env *protocol.Envelope, md metadata.MD) {
	if env == nil || md == nil {
		return
	}
	if version := firstMetadataValue(md, protocolVersionMetadataKey); version != "" {
		env.Version = version
	}
	if messageID := firstMetadataValue(md, messageIDMetadataKey); messageID != "" {
		env.ID = messageID
	}
	if messageType := firstMetadataValue(md, messageTypeMetadataKey); messageType != "" {
		env.Type = protocol.MessageType(messageType)
	}
	if ttlRaw := firstMetadataValue(md, ttlMetadataKey); ttlRaw != "" {
		if ttl, err := strconv.Atoi(ttlRaw); err == nil {
			env.TTL = ttl
		}
	}
	if hopRaw := firstMetadataValue(md, hopCountMetadataKey); hopRaw != "" {
		if hopCount, err := strconv.Atoi(hopRaw); err == nil {
			env.HopCount = hopCount
		}
	}
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
