package main

import (
	"grpc-agent/cmd"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "grpc-agent",
		Short: "A grpc-based agent for remote command execution",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	rootCmd.AddCommand(cmd.InitCmd())
	rootCmd.AddCommand(cmd.JoinCmd())
	rootCmd.AddCommand(cmd.RemoteExecCmd())
	rootCmd.AddCommand(cmd.ExecCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
