package cmd

import (
	"github.com/spf13/cobra"
)

var componentsCmd = &cobra.Command{
	Use:   "components",
	Short: "Commands for inspecting components files.",
}

func init() {
	rootCmd.AddCommand(componentsCmd)
}
