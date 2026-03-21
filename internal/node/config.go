package node

import "time"

type Config struct {
	NodeName                string
	NodeAddr                string
	Transport               string
	MaxPeers                int
	DiscoveryInterval       time.Duration
	ServiceDiscoveryAddress string
	KnownPeerAddresses      []string
}
