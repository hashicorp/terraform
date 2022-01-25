package cmd

import (
	"io"
	"strings"

	"github.com/spf13/cobra"
	cobraDoc "github.com/spf13/cobra/doc"
)

var commandDocsCmd = &cobra.Command{
	Use:    "command-docs",
	Short:  "A temporary plumbing command for generating command docs.",
	Hidden: true,
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		generateCommandDocsRecursive(rootCmd, cmd.OutOrStdout())
	},
}

func generateCommandDocsRecursive(cmd *cobra.Command, w io.Writer) {
	if cmd.Name() == "completion" {
		return
	}

	cmd.DisableAutoGenTag = true
	cobraDoc.GenMarkdownCustom(cmd, cmd.OutOrStdout(), func(cmd string) string {
		return "#" + strings.ReplaceAll(strings.TrimSuffix(cmd, ".md"), "_", "-")
	})

	for _, child := range cmd.Commands() {
		generateCommandDocsRecursive(child, w)
	}
}

func init() {
	rootCmd.AddCommand(commandDocsCmd)
}
