package node

import (
	"context"
	"log"
	"time"
)

func (node *Node) DiscoverySweep(ctx context.Context) {
	node.PingAllNodes(ctx, "discovery-sweep")
}

func (node *Node) StartDiscoveryLoop(interval time.Duration) func() {
	if interval <= 0 {
		return func() {}
	}

	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				node.DiscoverySweep(ctx)
				cancel()
			case <-stop:
				return
			}
		}
	}()

	log.Printf("Discovery loop started (interval=%s)", interval)
	return func() {
		close(stop)
	}
}
