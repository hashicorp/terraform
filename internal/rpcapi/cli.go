// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/cli"
)

// CLICommand is a command initialization callback for use with
// github.com/hashicorp/cli, allowing Terraform's "package main" to
// jump straight into the RPC plugin server without any interference
// from the usual Terraform CLI machinery in package "command", which
// is irrelevant here because this RPC API exists to bypass the
// Terraform CLI layer as much as possible.
func CLICommandFactory(opts CommandFactoryOpts) func() (cli.Command, error) {
	return func() (cli.Command, error) {
		return cliCommand{opts}, nil
	}
}

type CommandFactoryOpts struct {
	ExperimentsAllowed bool
	ShutdownCh         <-chan struct{}
}

type cliCommand struct {
	opts CommandFactoryOpts
}

// Help implements cli.Command.
func (c cliCommand) Help() string {
	helpText := `
Usage: terraform [global options] rpcapi

  Starts a gRPC server for programmatic access to Terraform Core from
  wrapping automation.

  This interface is currently intended only for HCP Terraform and is
  subject to breaking changes even in patch releases. Do not use this.
`
	return strings.TrimSpace(helpText)
}

// Run implements cli.Command.
func (c cliCommand) Run(args []string) int {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "This command does not accept any arguments.\n")
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// We'll adapt the caller's "shutdown channel" into a context
		// cancellation.
		for {
			select {
			case <-c.opts.ShutdownCh:
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()

	err := ServePlugin(ctx, ServerOpts{
		ExperimentsAllowed: c.opts.ExperimentsAllowed,
	})
	if err != nil {
		if err == ErrNotPluginClient {
			// TODO:
			//
			// The following message says that this interface is for HCP
			// Terraform only because we're using HCP Terraform's integration
			// with it to try to prove out the API/protocol design. By focusing
			// only on HCP Terraform as a client first, we can accommodate
			// any necessary breaking changes by ensuring that HCP Terraform's
			// client is updated before releasing an updated RPC API server
			// implementation.
			//
			// However, in the long run this should ideally become a documented
			// public interface with compatibility guarantees, at which point
			// we should change this error message only to express that this
			// is a machine-oriented integration API rather than something for
			// end-users to use directly. For example, the RPC server is likely
			// to make a better integration point for tools like the
			// Terraform Language Server in future too, assuming it grows to
			// include language analysis features.
			fmt.Fprintf(
				os.Stderr,
				`
This subcommand is for use by HCP Terraform and is not intended for direct use.
Its behavior is not subject to Terraform compatibility promises. To interact
with Terraform using the CLI workflow, refer to the main set of subcommands by
running the following command:
    terraform help

`)
		} else {
			fmt.Fprintf(os.Stderr, "Failed to start RPC server: %s.\n", err)
		}
		return 1
	}

	// NOTE: In practice it's impossible to get here, because if ServePlugin
	// doesn't error then it blocks forever and then eventually terminates
	// the process itself without returning.

	return 0
}

// Synopsis implements cli.Command.
func (c cliCommand) Synopsis() string {
	return "An RPC server used for integration with wrapping automation"
}
