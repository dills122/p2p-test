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

		peerCtx, cancel := context.WithTimeout(ctx, time.Second*3)
		reply, discovered, err := node.transport.Ping(peerCtx, peerAddr, node.Addr, envelope, msg)
		cancel()
		if err != nil {
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
		node.markPeerHealthy(peerAddr)
		node.mergePeerAddresses(discovered)
		queue = append(queue, discovered...)
		node.emitEvent(Event{
			Type:      EventTypeSent,
			MessageID: messageID,
			Peer:      peerAddr,
			Message:   msg,
			Timestamp: time.Now(),
		})
		log.Printf("Pinged node %s and got a status of %d", peerAddr, reply.Status)
	}
}

func (node *Node) PingOtherNode(peerAddr *string, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	envelope := protocol.NewEnvelope(protocol.MessageTypePing, node.Addr, []byte(message), protocol.DefaultTTL)
	if err := envelope.Validate(); err != nil {
		log.Fatalf("Failed to build ping envelope: %v", err)
	}
	messageID := envelope.ID
	pingReply, discovered, err := node.transport.Ping(ctx, *peerAddr, node.Addr, envelope, message)
	if err != nil {
		log.Fatalf("Failed to get status ping: %v", err)
	}
	node.markPeerHealthy(*peerAddr)
	node.mergePeerAddresses(discovered)
	node.emitEvent(Event{
		Type:      EventTypeSent,
		MessageID: messageID,
		Peer:      *peerAddr,
		Message:   message,
		Timestamp: time.Now(),
	})
	fmt.Printf("Reply received from node %s with status: %d and message: %s \n", *peerAddr, pingReply.Status, pingReply.Message)
}

func (node *Node) mergePeerAddresses(addresses []string) {
	for _, addr := range addresses {
		if err := node.AddPeer(addr); err != nil {
			continue
		}
	}
}
