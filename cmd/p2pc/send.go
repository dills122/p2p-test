package cmd

import (
	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "send a message in the network",
	Long:  `send a message in the network`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO implement the ping node here using existing node connection
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)

	sendCmd.Flags().StringP("message", "m", "Hello world!", "message to broadcast in test")
	sendCmd.MarkFlagRequired("message")
}
