package command

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/jsonrpc2"
	"github.com/hashicorp/terraform/lang/langserver"
	"github.com/hashicorp/terraform/tfdiags"
)

// LangServerCommand is a Command implementation that runs a Language Server
// that can provide information to a text editor or other development tool
// about the configuration rooted in the current working directory.
type LangServerCommand struct {
	Meta
}

func (c *LangServerCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("langserver")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var diags tfdiags.Diagnostics

	langserver.Run(
		context.Background(),
		configPath,
		jsonrpc2.NewHeaderStream(os.Stdin, os.Stdout),
	)

	c.showDiagnostics(diags)
	return 0
}

func (c *LangServerCommand) Help() string {
	helpText := `
Usage: terraform langserver [options] [DIR]

  Starts a language server to expose information about the configuration rooted
  in the current working directory.
`
	return strings.TrimSpace(helpText)
}

func (c *LangServerCommand) Synopsis() string {
	return "Language server protocol implementation"
}
