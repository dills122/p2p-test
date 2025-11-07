package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
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
	shellPrompt             = "$ "
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start node with interactive shell",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		setupCloseHandler()
		promptCtl, restoreLogging := setupPromptLogging(shellPrompt)
		defer restoreLogging()
		config := setupNodeConfig(cmd)
		activeNodeOne := node.New(config)

		go activeNodeOne.Start()

		isReady := activeNodeOne.CheckIfReady()
		if !isReady {
			log.Fatalf("Error when checking status of server")
		}
		reader := bufio.NewReader(os.Stdin)
		for {
			promptCtl.PrintIfNeeded()
			cmdString, err := reader.ReadString('\n')
			promptCtl.MarkPending()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			runCommand(cmdString, activeNodeOne)
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
	case "add-peer":
		if len(commandParts) < 2 {
			fmt.Println("Usage: add-peer <host:port>")
			return
		}
		if err := node.AddPeer(strings.TrimSpace(commandParts[1])); err != nil {
			fmt.Println(err)
		}
	case "peers":
		for _, peer := range node.ListPeers() {
			fmt.Printf("%s\t%s\t%s\n", peer.Addr, peer.Status, peer.LastSeen.Format(time.RFC3339))
		}
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

func setupPromptLogging(prompt string) (*promptState, func()) {
	originalWriter := log.Writer()
	promptCtl := newPromptState(prompt, os.Stdout)
	pWriter := newPromptWriter(promptCtl, originalWriter)
	log.SetOutput(pWriter)
	return promptCtl, func() {
		log.SetOutput(originalWriter)
	}
}

type promptWriter struct {
	mu        sync.Mutex
	state     *promptState
	logWriter io.Writer
}

func newPromptWriter(state *promptState, logWriter io.Writer) *promptWriter {
	return &promptWriter{
		state:     state,
		logWriter: logWriter,
	}
}

func (p *promptWriter) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, err := fmt.Fprint(p.state.out, "\r"); err != nil {
		return 0, err
	}
	p.state.MarkPending()
	if _, err := p.logWriter.Write(b); err != nil {
		return 0, err
	}
	p.state.PrintNow()
	return len(b), nil
}

type promptState struct {
	mu      sync.Mutex
	prompt  string
	out     io.Writer
	pending bool
}

func newPromptState(prompt string, out io.Writer) *promptState {
	return &promptState{prompt: prompt, out: out, pending: true}
}

func (p *promptState) PrintIfNeeded() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.pending {
		return
	}
	fmt.Fprint(p.out, p.prompt)
	p.pending = false
}

func (p *promptState) PrintNow() {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprint(p.out, p.prompt)
	p.pending = false
}

func (p *promptState) MarkPending() {
	p.mu.Lock()
	p.pending = true
	p.mu.Unlock()
}
