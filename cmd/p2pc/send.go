/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

// sendCmd represents the pingTest command
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "send a message in the network",
	Long:  `send a message in the network`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	rootCmd.AddCommand(sendCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pingTestCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	sendCmd.Flags().StringP("message", "m", "Hello world!", "message to broadcast in test")
	sendCmd.MarkFlagRequired("message")
}
