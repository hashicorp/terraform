package cmd

import (
	"context"
	"os"

	"github.com/hashicorp/terraform/internal/rpcapi"
	"github.com/hashicorp/terraform/internal/terraform"

	"github.com/spf13/cobra"
)

var coreRPCCmd = &cobra.Command{
	Use:   "core-rpc",
	Short: "Run as an RPC server exposing Terraform Core.",
	Long: `Runs a gRPC-based RPC server exposing Terraform Core functionality directly.
	
This is an internal plumbing command that most users should not need to use directly.`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()

		if !rpcapi.RunningAsPlugin(ctx) {
			cmd.PrintErrln("The core-rpc command is only for use by other wrapper programs that understand its RPC protocol.")
			os.Exit(1)
		}

		err := rpcapi.Serve(ctx, rpcapi.ServeOpts{
			GetCoreOpts: func() *terraform.ContextOpts {
				// TODO: We need to actually return some suitable options here,
				// so that Terraform Core will be able to launch providers, etc.
				return nil
			},
		})
		if err != nil {
			cmd.PrintErrf("Failed to launch RPC server: %s\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(coreRPCCmd)
}
