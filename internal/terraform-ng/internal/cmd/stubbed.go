package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func stubbedCommand(cmd *cobra.Command, args []string) {
	cmd.PrintErrf(
		"The %q command is currently only a stub, as a part of exploring possible CLI command heirarchies.\n",
		cmd.CommandPath(),
	)

	os.Exit(1)
}
