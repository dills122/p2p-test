package node

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	ping "github.com/dills122/p2p-test/pkg/ping"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
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
	service.trackCaller(ctx)
	service.sendPeerMetadata(ctx)
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
	options                   []grpc.ServerOption
	enabledReflection         bool
	shutdownHook              func()
	enabledHealthCheck        bool
	disableDefaultHealthCheck bool
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

func (service *Service) trackCaller(ctx context.Context) {
	if service.node == nil {
		return
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
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
	}
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
