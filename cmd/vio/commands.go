package vio

import "github.com/spf13/cobra"

func SpawnAgentCmd() *cobra.Command {
	cmd := cobra.Command {
		Use: "vio",
		Short: "Start the vio agent in a new terminal session",
		Run: SpawnAgent
	}
}

func SpawnAgent(cmd *cobra.Command, args []string) {
	// spawn a new terminal session and start the agent here.
} 