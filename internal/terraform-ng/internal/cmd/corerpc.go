package cmd

import (
	"github.com/spf13/cobra"
)

var coreRPCCmd = &cobra.Command{
	Use:   "core-rpc",
	Short: "Run as an RPC server exposing Terraform Core.",
	Long: `Runs a gRPC-based RPC server exposing Terraform Core functionality directly.
	
This is an internal plumbing command that most users should not need to use directly.`,
	Hidden: true,
	Run:    stubbedCommand,
}

func init() {
	rootCmd.AddCommand(coreRPCCmd)
}
