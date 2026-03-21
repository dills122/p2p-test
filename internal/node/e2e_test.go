package node

import (
	"context"
	"net"
	"testing"
	"time"

	ping "github.com/dills122/p2p-test/pkg/ping"
	"github.com/dills122/p2p-test/pkg/protocol"
	"google.golang.org/grpc"
)

func TestTransportPingE2E(t *testing.T) {
	receiver := New(Config{
		NodeName: "receiver",
		NodeAddr: freeAddress(t),
	})
	receiverStop := startExistingNodeServer(t, receiver)
	defer receiverStop()

	sender := New(Config{
		NodeName: "sender",
		NodeAddr: "127.0.0.1:19999",
	})

	recvEvents := make(chan Event, 4)
	unsub := receiver.Subscribe(func(evt Event) {
		if evt.Type == EventTypeRecv {
			recvEvents <- evt
		}
	})
	defer unsub()

	env := protocol.NewEnvelope(protocol.MessageTypePing, sender.Addr, []byte("hello"), protocol.DefaultTTL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	reply, _, err := sender.transport.Ping(ctx, receiver.Addr, sender.Addr, sender.publicKeyBase64(), env, "hello")
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if reply.Status != int32(READY) {
		t.Fatalf("expected READY status, got %d", reply.Status)
	}

	select {
	case evt := <-recvEvents:
		if evt.MessageID != env.ID {
			t.Fatalf("expected message id %s, got %s", env.ID, evt.MessageID)
		}
		if evt.Message != "hello" {
			t.Fatalf("expected message hello, got %q", evt.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for receive event")
	}
}

func TestTransportPingDuplicateDroppedE2E(t *testing.T) {
	receiver := New(Config{
		NodeName: "receiver",
		NodeAddr: freeAddress(t),
	})
	receiverStop := startExistingNodeServer(t, receiver)
	defer receiverStop()

	sender := New(Config{
		NodeName: "sender",
		NodeAddr: "127.0.0.1:19998",
	})

	recvEvents := make(chan Event, 8)
	unsub := receiver.Subscribe(func(evt Event) {
		if evt.Type == EventTypeRecv {
			recvEvents <- evt
		}
	})
	defer unsub()

	env := protocol.NewEnvelope(protocol.MessageTypePing, sender.Addr, []byte("duplicate"), protocol.DefaultTTL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	firstReply, _, err := sender.transport.Ping(ctx, receiver.Addr, sender.Addr, sender.publicKeyBase64(), env, "duplicate")
	if err != nil {
		t.Fatalf("first ping failed: %v", err)
	}
	secondReply, _, err := sender.transport.Ping(ctx, receiver.Addr, sender.Addr, sender.publicKeyBase64(), env, "duplicate")
	if err != nil {
		t.Fatalf("second ping failed: %v", err)
	}

	if firstReply.Status != int32(READY) {
		t.Fatalf("expected first status READY, got %d", firstReply.Status)
	}
	if secondReply.Status != int32(OFFLINE) {
		t.Fatalf("expected second status OFFLINE for duplicate, got %d", secondReply.Status)
	}

	select {
	case <-recvEvents:
	case <-time.After(2 * time.Second):
		t.Fatal("expected first receive event")
	}

	select {
	case evt := <-recvEvents:
		t.Fatalf("expected duplicate to be dropped, got extra event %+v", evt)
	case <-time.After(300 * time.Millisecond):
		// No second event is expected.
	}
}

func startExistingNodeServer(t *testing.T, node *Node) func() {
	t.Helper()
	lis, err := net.Listen("tcp", node.Addr)
	if err != nil {
		t.Fatalf("listen failed on %s: %v", node.Addr, err)
	}

	srv := grpc.NewServer()
	ping.RegisterPingServiceServer(srv, &Service{node: node})

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(lis)
	}()

	return func() {
		srv.GracefulStop()
		_ = lis.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for server to stop")
		}
	}
}

func freeAddress(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate address: %v", err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()
	return addr
}
