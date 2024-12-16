package node

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	ping "github.com/dills122/p2p-test/pkg/ping"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	OFFLINE int = 0
	READY   int = 1
	CLOSED  int = 2
)

type Service struct {
	ping.UnimplementedPingServiceServer
	grpc_health_v1.UnimplementedHealthServer
}

func (service *Service) PingNode(ctx context.Context, stream *ping.PingRequest) (*ping.PingReply, error) {
	return &ping.PingReply{Message: stream.Message, Status: int32(READY)}, nil
}

func StartServer(addr string) {
	log.Println("Started gRPC Server")
	builder := GrpcServerBuilder{}
	s := builder.Build()
	err := s.Start(addr)
	if err != nil {
		log.Fatalf("%v", err)
	}
	s.AwaitTermination(func() {
		log.Println("Shutting down the server")
	})
}

func setupServerServices(server *grpc.Server) {
	ping.RegisterPingServiceServer(server, &Service{})
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

func (sb *GrpcServerBuilder) Build() GrpcServer {
	srv := grpc.NewServer(sb.options...)
	setupServerServices(srv)
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
