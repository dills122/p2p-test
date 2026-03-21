package node

import (
	"fmt"

	"github.com/dills122/p2p-test/pkg/protocol"
)

func (node *Node) acceptIncomingEnvelope(env protocol.Envelope) error {
	if err := env.Validate(); err != nil {
		return err
	}
	if node.seen.IsDuplicate(env.ID) {
		return fmt.Errorf("duplicate message id %q", env.ID)
	}
	if env.TTL <= 0 {
		return fmt.Errorf("expired ttl for message id %q", env.ID)
	}
	return nil
}
