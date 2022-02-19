package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	commCmd "github.com/dills122/p2p-test/cmd/p2pc"
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
			runCommand(cmdString)
		}
	},
}

func runCommand(commandStr string) {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr := strings.Fields(commandStr)
	if len(arrCommandStr) <= 0 {
		return
	}
	switch arrCommandStr[0] {
	case "exit":
		os.Exit(0)
	case "send":
		commCmd.Execute()
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
