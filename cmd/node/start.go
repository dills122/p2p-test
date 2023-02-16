package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dills122/p2p-test/node"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start node with interactive shell",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		setupCloseHandler()
		config := setupNodeConfig(cmd)
		activeNodeOne := node.New(config.NodeName, config.NodeAddr)

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
	var listenerAddress string
	if len(listenerAddressSlice) > 0 {
		listenerAddress = listenerAddressSlice[0]
	} else {
		listenerAddress = "127.0.0.1:80000"
	}
	config := node.Config{
		NodeName:                nodeName,
		NodeAddr:                nodeAddress,
		ServiceDiscoveryAddress: listenerAddress,
	}
	return config
}

func init() {
	rootCmd.AddCommand(startCmd)

	id := uuid.New()

	startCmd.Flags().StringP("address", "a", "", "address for node")
	startCmd.MarkFlagRequired("address")
	startCmd.Flags().StringP("name", "n", id.String(), "name for node")
	startCmd.Flags().StringSliceP("listener-addresses", "l", []string{}, "list of known rely nodes")
}
