package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/hcl/fmtcmd"
	"github.com/mitchellh/cli"
)

const (
	fileExtension = "tf"
)

// FmtCommand is a Command implementation that rewrites Terraform config
// files to a canonical format and style.
type FmtCommand struct {
	Meta
	opts fmtcmd.Options
}

func (c *FmtCommand) Run(args []string) int {
	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("fmt", flag.ContinueOnError)
	cmdFlags.BoolVar(&c.opts.List, "list", false, "list")
	cmdFlags.BoolVar(&c.opts.Write, "write", false, "write")
	cmdFlags.BoolVar(&c.opts.Diff, "diff", false, "diff")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		c.Ui.Error("The fmt command expects no arguments.")
		cmdFlags.Usage()
		return 1
	}

	dir := "."
	output := &cli.UiWriter{Ui: c.Ui}
	err := fmtcmd.Run([]string{dir}, []string{fileExtension}, nil, output, c.opts)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error running fmt: %s", err))
		return 2
	}

	return 0
}

func (c *FmtCommand) Help() string {
	helpText := `
Usage: terraform fmt [options]

	Rewrites all Terraform configuration files in the current working
	directory to a canonical format.

Options:

  -list            List files whose formatting differs

  -write           Write result to source file instead of STDOUT

  -diff            Display diffs instead of rewriting files

`
	return strings.TrimSpace(helpText)
}

func (c *FmtCommand) Synopsis() string {
	return "Rewrites config files to canonical format"
}
