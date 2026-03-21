package protocol

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	CurrentVersion = "1"
	DefaultTTL     = 4
)

type MessageType string

const (
	MessageTypePing          MessageType = "PING"
	MessageTypePeerAnnounce  MessageType = "PEER_ANNOUNCE"
	MessageTypeTx            MessageType = "TX"
	MessageTypeBlockProposal MessageType = "BLOCK_PROPOSAL"
	MessageTypeVote          MessageType = "VOTE"
)

type Envelope struct {
	Version   string
	ID        string
	Type      MessageType
	Origin    string
	Timestamp time.Time
	TTL       int
	HopCount  int
	Payload   []byte
	Signature []byte
}

func NewEnvelope(messageType MessageType, origin string, payload []byte, ttl int) Envelope {
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	return Envelope{
		Version:   CurrentVersion,
		ID:        uuid.NewString(),
		Type:      messageType,
		Origin:    strings.TrimSpace(origin),
		Timestamp: time.Now().UTC(),
		TTL:       ttl,
		HopCount:  0,
		Payload:   payload,
	}
}

func (e Envelope) Validate() error {
	if strings.TrimSpace(e.Version) == "" {
		return errors.New("envelope version is required")
	}
	if e.Version != CurrentVersion {
		return fmt.Errorf("unsupported envelope version: %s", e.Version)
	}
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("envelope id is required")
	}
	if strings.TrimSpace(string(e.Type)) == "" {
		return errors.New("envelope type is required")
	}
	if strings.TrimSpace(e.Origin) == "" {
		return errors.New("envelope origin is required")
	}
	if e.Timestamp.IsZero() {
		return errors.New("envelope timestamp is required")
	}
	if e.TTL <= 0 {
		return errors.New("envelope ttl must be greater than zero")
	}
	if e.HopCount < 0 {
		return errors.New("envelope hop count cannot be negative")
	}
	return nil
}

func (e *Envelope) DecrementTTL() bool {
	if e == nil {
		return false
	}
	e.TTL--
	e.HopCount++
	return e.TTL > 0
}
