/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"time"

	"github.com/dills122/p2p-test/node"
	"github.com/spf13/cobra"
)

// pingTestCmd represents the pingTest command
var pingTestCmd = &cobra.Command{
	Use:   "pingTest",
	Short: "A ping test",
	Long:  `A test that will start x number of nodes and ping each with a desired message before shutting down`,
	Run: func(cmd *cobra.Command, args []string) {
		confNodeOne := node.Config{
			NodeName:                "node-one",
			NodeAddr:                "127.0.0.1:10000",
			ServiceDiscoveryAddress: "127.0.0.1:80000",
		}
		fmt.Printf("Node: %s started at %s and running on %s", confNodeOne.NodeName, time.Now().UTC(), confNodeOne.NodeAddr)
		activeNodeOne := node.New(confNodeOne.NodeName, confNodeOne.NodeAddr)
		activeNodeOne.Start()
	},
}

func init() {
	rootCmd.AddCommand(pingTestCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pingTestCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	pingTestCmd.Flags().StringP("message", "m", "Hello world!", "message to broadcast in test")
}
