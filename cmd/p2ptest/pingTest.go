package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/dills122/p2p-test/internal/node"
	"github.com/spf13/cobra"
)

var pingTestCmd = &cobra.Command{
	Use:   "pingTest",
	Short: "A ping test",
	Long:  `A test that will start x number of nodes and ping each with a desired message before shutting down`,
	Run: func(cmd *cobra.Command, args []string) {
		confNodeOne := node.Config{
			NodeName:                "node-one",
			NodeAddr:                "127.0.0.1:10000",
			Transport:               node.TransportGRPC,
			ServiceDiscoveryAddress: "127.0.0.1:10000",
			KnownPeerAddresses:      []string{"127.0.0.1:10001"},
		}
		activeNodeOne := node.New(confNodeOne)
		go activeNodeOne.Start()
		fmt.Printf("Node: %s started at %s and running on %s \n", confNodeOne.NodeName, time.Now().UTC(), confNodeOne.NodeAddr)
		confNodeTwo := node.Config{
			NodeName:                "node-two",
			NodeAddr:                "127.0.0.1:10001",
			Transport:               node.TransportGRPC,
			ServiceDiscoveryAddress: "127.0.0.1:10001",
			KnownPeerAddresses:      []string{"127.0.0.1:10000"},
		}
		activeNodeTwo := node.New(confNodeTwo)
		go activeNodeTwo.Start()
		fmt.Printf("Node: %s started at %s and running on %s \n", confNodeTwo.NodeName, time.Now().UTC(), confNodeTwo.NodeAddr)
		time.Sleep(2 * time.Second)
		message, _ := cmd.Flags().GetString("message")
		if err := activeNodeOne.PingOtherNode(confNodeTwo.NodeAddr, message+" from node 1"); err != nil {
			log.Printf("node one ping failed: %v", err)
		}
		if err := activeNodeTwo.PingOtherNode(confNodeOne.NodeAddr, message+" from node 2"); err != nil {
			log.Printf("node two ping failed: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(pingTestCmd)

	pingTestCmd.Flags().StringP("message", "m", "Hello world!", "message to broadcast in test")
}
