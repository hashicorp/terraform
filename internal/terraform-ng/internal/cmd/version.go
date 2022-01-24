package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the Terraform-NG version number",
	Run: func(cmd *cobra.Command, args []string) {
		versionStr := "devel"
		if info, ok := debug.ReadBuildInfo(); ok {
			versionStr = info.Main.Version
		}
		fmt.Printf("Terraform-NG %s\n", versionStr)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
