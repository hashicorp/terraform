package cmd

import (
	"github.com/spf13/cobra"
)

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Commands for module developers.",
}

func init() {
	rootCmd.AddCommand(moduleCmd)
}
