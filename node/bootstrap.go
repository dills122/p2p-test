package node

import (
	"log"
	"net"
	"strings"
)

var defaultPeerAddresses = []string{
	"127.0.0.1:10000",
	"127.0.0.1:10001",
	"127.0.0.1:10002",
	"127.0.0.1:10003",
}

func sanitizeBootstrap(selfAddr string, addresses []string) []string {
	if len(addresses) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{})
	var peers []string
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr == "" || addr == selfAddr {
			continue
		}
		if _, _, err := net.SplitHostPort(addr); err != nil {
			log.Printf("Skipping peer address %q: %v", addr, err)
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		peers = append(peers, addr)
		seen[addr] = struct{}{}
	}
	return peers
}
