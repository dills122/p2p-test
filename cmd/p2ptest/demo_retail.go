package cmd

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/dills122/p2p-test/internal/node"
	"github.com/spf13/cobra"
)

var demoRetailCmd = &cobra.Command{
	Use:   "demoRetail",
	Short: "run a real-world payment network demo",
	Long:  `Starts multiple nodes that model a payment network and runs a scripted scenario (bootstrap, transaction fanout, outage/backoff, and late-node join).`,
	Run: func(cmd *cobra.Command, args []string) {
		basePort, _ := cmd.Flags().GetInt("base-port")
		transport, _ := cmd.Flags().GetString("transport")

		storefrontAddr := fmt.Sprintf("127.0.0.1:%d", basePort)
		gatewayAddr := fmt.Sprintf("127.0.0.1:%d", basePort+1)
		riskAddr := fmt.Sprintf("127.0.0.1:%d", basePort+2)
		settlementAddr := fmt.Sprintf("127.0.0.1:%d", basePort+3)
		analyticsAddr := fmt.Sprintf("127.0.0.1:%d", basePort+4)

		storefront := node.New(node.Config{
			NodeName:           "storefront-us-east",
			NodeAddr:           storefrontAddr,
			Transport:          transport,
			KnownPeerAddresses: []string{gatewayAddr},
			MaxPeers:           128,
		})
		gateway := node.New(node.Config{
			NodeName:           "gateway-us-east",
			NodeAddr:           gatewayAddr,
			Transport:          transport,
			KnownPeerAddresses: []string{storefrontAddr, riskAddr},
			MaxPeers:           128,
		})
		risk := node.New(node.Config{
			NodeName:           "risk-us-east",
			NodeAddr:           riskAddr,
			Transport:          transport,
			KnownPeerAddresses: []string{gatewayAddr, settlementAddr},
			MaxPeers:           128,
		})
		settlement := node.New(node.Config{
			NodeName:           "settlement-us-east",
			NodeAddr:           settlementAddr,
			Transport:          transport,
			KnownPeerAddresses: []string{riskAddr},
			MaxPeers:           128,
		})

		nodes := []*node.Node{storefront, gateway, risk, settlement}
		for _, n := range nodes {
			go n.Start()
		}
		waitForNodesReady(nodes)

		log.Println("phase=bootstrap msg=\"warming peer graph\"")
		mustPing(storefront, gatewayAddr, "AUTH_REQUEST order=8842 amount=129.99 customer=42")
		mustPing(gateway, riskAddr, "RISK_CHECK order=8842")
		mustPing(risk, settlementAddr, "SETTLEMENT_PREPARE order=8842")

		log.Println("phase=fanout msg=\"broadcasting transaction event from storefront\"")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		storefront.PingAllNodes(ctx, "TX order=8842 status=AUTHORIZED")
		cancel()
		time.Sleep(400 * time.Millisecond)

		log.Println("phase=outage msg=\"simulating unreachable settlement endpoint to show backoff\"")
		_ = storefront.AddPeer(fmt.Sprintf("127.0.0.1:%d", basePort+40))
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			storefront.PingAllNodes(ctx, fmt.Sprintf("HEALTH_CHECK seq=%d", i+1))
			cancel()
			time.Sleep(250 * time.Millisecond)
		}

		log.Println("phase=late_join msg=\"starting analytics node and re-running discovery\"")
		analytics := node.New(node.Config{
			NodeName:           "analytics-us-west",
			NodeAddr:           analyticsAddr,
			Transport:          transport,
			KnownPeerAddresses: []string{gatewayAddr},
			MaxPeers:           128,
		})
		go analytics.Start()
		waitForNodesReady([]*node.Node{analytics})

		mustPing(analytics, gatewayAddr, "ANALYTICS_HELLO region=us-west")
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		gateway.PingAllNodes(ctx, "PEER_RECONCILE reason=analytics_join")
		cancel()
		time.Sleep(500 * time.Millisecond)

		printPeerSnapshot("storefront", storefront.ListPeers())
		printPeerSnapshot("gateway", gateway.ListPeers())
		printPeerSnapshot("risk", risk.ListPeers())
		printPeerSnapshot("settlement", settlement.ListPeers())
		printPeerSnapshot("analytics", analytics.ListPeers())

		log.Println("demo_complete=true")
	},
}

func init() {
	rootCmd.AddCommand(demoRetailCmd)
	demoRetailCmd.Flags().Int("base-port", 12000, "base port for demo node addresses")
	demoRetailCmd.Flags().String("transport", node.TransportGRPC, "transport backend (grpc|libp2p)")
}

func waitForNodesReady(nodes []*node.Node) {
	deadline := time.Now().Add(8 * time.Second)
	for _, n := range nodes {
		for {
			isReady, err := n.CheckIfReady()
			if err == nil && isReady {
				break
			}
			if time.Now().After(deadline) {
				log.Fatalf("node %s did not become ready in time (err=%v)", n.Name, err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func mustPing(n *node.Node, addr string, message string) {
	if err := n.PingOtherNode(addr, message); err != nil {
		log.Fatalf("demo ping failed from %s to %s: %v", n.Name, addr, err)
	}
}

func printPeerSnapshot(name string, peers []node.Peer) {
	sort.Slice(peers, func(i, j int) bool { return peers[i].Addr < peers[j].Addr })
	log.Printf("snapshot node=%s peers=%d", name, len(peers))
	for _, peer := range peers {
		log.Printf("snapshot node=%s peer=%s status=%s score=%d failures=%d rtt=%s cooldown=%s",
			name,
			peer.Addr,
			peer.Status,
			peer.Score,
			peer.Failures,
			peer.LastRTT.String(),
			peer.CooldownUntil.Format(time.RFC3339),
		)
	}
}
