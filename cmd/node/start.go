/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start node with interactive shell",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("start called")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	id := uuid.New()

	startCmd.Flags().StringP("address", "a", "", "address for node")
	startCmd.MarkFlagRequired("address")
	startCmd.Flags().StringP("name", "n", id.String(), "name for node")
	startCmd.Flags().StringSliceP("listener-addresses", "l", []string{}, "list of known rely nodes")
}
