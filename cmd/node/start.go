package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dills122/p2p-test/node"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	defaultNodeAddress      = "127.0.0.1:10002"
	defaultListenerAddress  = "127.0.0.1:10000"
	sendCommandUsageMessage = "Usage: send <message>"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start node with interactive shell",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		setupCloseHandler()
		config := setupNodeConfig(cmd)
		activeNodeOne := node.New(config)

		go activeNodeOne.Start()

		isReady := activeNodeOne.CheckIfReady()
		if !isReady {
			log.Fatalf("Error when checking status of server")
		}
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("$ ")
			cmdString, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			runCommand(cmdString, &activeNodeOne)
		}
	},
}

func runCommand(commandStr string, node *node.Node) {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	commandParts := strings.SplitN(commandStr, " ", 2)
	if len(commandParts) <= 0 {
		return
	}
	switch commandParts[0] {
	case "exit":
		os.Exit(0)
	case "send":
		if len(commandParts) < 2 {
			fmt.Println(sendCommandUsageMessage)
			return
		}
		msg := commandParts[1]
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		node.PingAllNodes(ctx, msg)
		defer cancel()
	default:
		fmt.Println("Unknown command")
	}
}

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Exiting interactive console")
		os.Exit(0)
	}()
}

func setupNodeConfig(cmd *cobra.Command) node.Config {
	nodeAddress, _ := cmd.Flags().GetString("address")
	nodeName, _ := cmd.Flags().GetString("name")
	listenerAddressSlice, _ := cmd.Flags().GetStringSlice("listener-addresses")
	peers, err := parsePeerAddresses(listenerAddressSlice)
	if err != nil {
		log.Fatalf("Invalid listener address: %v", err)
	}
	listenerAddress := defaultListenerAddress
	if len(peers) > 0 {
		listenerAddress = peers[0]
	}
	config := node.Config{
		NodeName:                nodeName,
		NodeAddr:                nodeAddress,
		ServiceDiscoveryAddress: listenerAddress,
		KnownPeerAddresses:      peers,
	}
	return config
}

func parsePeerAddresses(raw []string) ([]string, error) {
	peers := make([]string, 0, len(raw))
	for _, addr := range raw {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		if !strings.Contains(addr, ":") {
			return nil, fmt.Errorf("peer address %q is missing a port (expected host:port)", addr)
		}
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return nil, fmt.Errorf("peer address %q is invalid: %w", addr, err)
		}
		peers = append(peers, addr)
	}
	return peers, nil
}

func init() {
	rootCmd.AddCommand(startCmd)

	id := uuid.New()

	startCmd.Flags().StringP("address", "a", defaultNodeAddress, "address (host:port) to bind this node to")
	startCmd.Flags().StringP("name", "n", id.String(), "name for node")
	startCmd.Flags().StringSliceP("listener-addresses", "l", []string{defaultListenerAddress}, "list of known relay nodes")
}
