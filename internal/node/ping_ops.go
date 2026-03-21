package node

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dills122/p2p-test/pkg/protocol"
)

func (node *Node) PingAllNodes(ctx context.Context, msg string) {
	if len(msg) == 0 {
		msg = "Pinging"
	}
	envelope := protocol.NewEnvelope(protocol.MessageTypePing, node.Addr, []byte(msg), protocol.DefaultTTL)
	if err := envelope.Validate(); err != nil {
		log.Printf("failed to build ping envelope: %v", err)
		return
	}
	messageID := envelope.ID
	knownPeers := node.peers.List()
	if len(knownPeers) == 0 {
		log.Println("No known peers to ping")
		return
	}
	log.Println("Executing known peer list")

	visited := make(map[string]struct{})
	queue := make([]string, 0, len(knownPeers))
	for _, peer := range knownPeers {
		queue = append(queue, peer.Addr)
	}

	for len(queue) > 0 {
		peerAddr := queue[0]
		queue = queue[1:]
		if peerAddr == "" || peerAddr == node.Addr {
			continue
		}
		if _, ok := visited[peerAddr]; ok {
			continue
		}
		visited[peerAddr] = struct{}{}
		if allowed, wait := node.canAttemptPeer(peerAddr); !allowed {
			logNetworkEvent("ping_send", map[string]string{
				"msg_id":   messageID,
				"msg_type": string(envelope.Type),
				"peer":     peerAddr,
				"result":   "skipped_backoff",
				"wait_ms":  fmt.Sprintf("%d", wait.Milliseconds()),
			})
			continue
		}
		start := time.Now()

		peerCtx, cancel := context.WithTimeout(ctx, time.Second*3)
		reply, discovered, err := node.transport.Ping(peerCtx, peerAddr, node.Addr, envelope, msg)
		cancel()
		if err != nil {
			backoff := node.markPeerFailure(peerAddr)
			logNetworkEvent("ping_send", map[string]string{
				"latency_ms": fmt.Sprintf("%d", time.Since(start).Milliseconds()),
				"msg_id":     messageID,
				"msg_type":   string(envelope.Type),
				"peer":       peerAddr,
				"backoff_ms": fmt.Sprintf("%d", backoff.Milliseconds()),
				"result":     "error",
			})
			log.Printf("failed to ping node at address %s: %v", peerAddr, err)
			node.emitEvent(Event{
				Type:      EventTypeError,
				MessageID: messageID,
				Peer:      peerAddr,
				Message:   msg,
				Err:       err,
				Timestamp: time.Now(),
			})
			continue
		}
		node.markPeerHealthy(peerAddr, time.Since(start))
		node.mergePeerAddresses(discovered)
		queue = append(queue, discovered...)
		node.emitEvent(Event{
			Type:      EventTypeSent,
			MessageID: messageID,
			Peer:      peerAddr,
			Message:   msg,
			Timestamp: time.Now(),
		})
		logNetworkEvent("ping_send", map[string]string{
			"latency_ms": fmt.Sprintf("%d", time.Since(start).Milliseconds()),
			"msg_id":     messageID,
			"msg_type":   string(envelope.Type),
			"peer":       peerAddr,
			"result":     "ok",
			"status":     fmt.Sprintf("%d", reply.Status),
		})
		log.Printf("Pinged node %s and got a status of %d", peerAddr, reply.Status)
	}
}

func (node *Node) PingOtherNode(peerAddr string, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	envelope := protocol.NewEnvelope(protocol.MessageTypePing, node.Addr, []byte(message), protocol.DefaultTTL)
	if err := envelope.Validate(); err != nil {
		return fmt.Errorf("build ping envelope: %w", err)
	}
	messageID := envelope.ID
	if allowed, wait := node.canAttemptPeer(peerAddr); !allowed {
		return fmt.Errorf("peer %s in reconnect cooldown for %s", peerAddr, wait.Round(time.Millisecond))
	}
	start := time.Now()
	pingReply, discovered, err := node.transport.Ping(ctx, peerAddr, node.Addr, envelope, message)
	if err != nil {
		backoff := node.markPeerFailure(peerAddr)
		logNetworkEvent("ping_send", map[string]string{
			"latency_ms": fmt.Sprintf("%d", time.Since(start).Milliseconds()),
			"msg_id":     messageID,
			"msg_type":   string(envelope.Type),
			"peer":       peerAddr,
			"backoff_ms": fmt.Sprintf("%d", backoff.Milliseconds()),
			"result":     "error",
		})
		return fmt.Errorf("get status ping: %w", err)
	}
	node.markPeerHealthy(peerAddr, time.Since(start))
	node.mergePeerAddresses(discovered)
	node.emitEvent(Event{
		Type:      EventTypeSent,
		MessageID: messageID,
		Peer:      peerAddr,
		Message:   message,
		Timestamp: time.Now(),
	})
	logNetworkEvent("ping_send", map[string]string{
		"latency_ms": fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		"msg_id":     messageID,
		"msg_type":   string(envelope.Type),
		"peer":       peerAddr,
		"result":     "ok",
		"status":     fmt.Sprintf("%d", pingReply.Status),
	})
	fmt.Printf("Reply received from node %s with status: %d and message: %s \n", peerAddr, pingReply.Status, pingReply.Message)
	return nil
}

func (node *Node) mergePeerAddresses(addresses []string) {
	for _, addr := range addresses {
		if err := node.AddPeer(addr); err != nil {
			continue
		}
	}
}
