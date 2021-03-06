// tfexprtest is a Terraform Core development tool for working with the
// expression stress-testing helper functions.
//
// This is a command-line program intended to be run separately in development
// environments, not functionality embedded in the main "terraform" executables.
// In most cases, you can conveniently run this tool without a separate
// compilation step using the following:
//
//     go run github.com/hashicorp/terraform/internal/exprstress/tfexprstress
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/hashicorp/terraform/internal/terminal"
)

func main() {
	streams, err := terminal.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize terminal: %s\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		streams.Eprint(usageHelp)
		os.Exit(1)
	}

	if args[0] == "help" || args[0] == "-help" || args[0] == "--help" || args[0] == "-h" {
		streams.Print(usageHelp)
		os.Exit(0)
	}

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	switch cmd, args := args[0], args[1:]; cmd {
	case "run":
		os.Exit(runCommand(ctx, streams, args))
	default:
		streams.Eprintf("Error: Unsupported command %q\n\n", cmd)
		streams.Eprint(usageHelp)
		os.Exit(1)
	}
}

const usageHelp string = `Usage: tfexprstress <subcommand> [arguments...]

Subcommands:
  run    Run indefinitely (until interrupted) generating and testing
         random expressions.
`
