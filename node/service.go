package node

import (
	"context"
	"log"
	"net"

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
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	setupServerServices(grpcServer)

	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func setupServerServices(server *grpc.Server) {
	ping.RegisterPingServiceServer(server, &Service{})
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())
}
