package node

import (
	"context"
	"io"
	"log"

	"google.golang.org/grpc/health/grpc_health_v1"
)

func (node *Node) CheckIfReady() bool {
	conn, err := node.transport.Dial(node.Addr)
	if err != nil {
		log.Fatalf("Unable to connect to health service on %s: %v", node.Addr, err)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
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
				done <- false // stream finished
				return
			}
			if err != nil {
				log.Fatalf("cannot receive %v", err)
			}
			log.Printf("Resp received: %s", resp.Status)
			if resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
				done <- true
				return
			}
		}
	}()

	isAvailable := <-done
	return isAvailable
}
