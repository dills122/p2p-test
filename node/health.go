package node

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc/health/grpc_health_v1"
)

func (node *Node) CheckIfReady() (bool, error) {
	conn, err := node.transport.Dial(node.Addr)
	if err != nil {
		return false, fmt.Errorf("unable to connect to health service on %s: %w", node.Addr, err)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
	ctx := context.Background()
	stream, err := client.Watch(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return false, fmt.Errorf("open health stream: %w", err)
	}

	done := make(chan bool, 1)
	errCh := make(chan error, 1)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- false // stream finished
				return
			}
			if err != nil {
				errCh <- fmt.Errorf("receive health stream response: %w", err)
				return
			}
			if resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
				done <- true
				return
			}
		}
	}()

	select {
	case isAvailable := <-done:
		return isAvailable, nil
	case err := <-errCh:
		return false, err
	}
}
