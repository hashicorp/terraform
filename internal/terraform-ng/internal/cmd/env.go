package cmd

import (
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Commands for manipulating environments.",
}

func init() {
	rootCmd.AddCommand(envCmd)
}
