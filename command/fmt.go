package command

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/hcl/hcl/fmtcmd"
	"github.com/mitchellh/cli"
)

const (
	stdinArg      = "-"
	fileExtension = "tf"
)

// FmtCommand is a Command implementation that rewrites Terraform config
// files to a canonical format and style.
type FmtCommand struct {
	Meta
	opts  fmtcmd.Options
	check bool
	input io.Reader // STDIN if nil
}

func (c *FmtCommand) Run(args []string) int {
	if c.input == nil {
		c.input = os.Stdin
	}

	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := flag.NewFlagSet("fmt", flag.ContinueOnError)
	cmdFlags.BoolVar(&c.opts.List, "list", true, "list")
	cmdFlags.BoolVar(&c.opts.Write, "write", true, "write")
	cmdFlags.BoolVar(&c.opts.Diff, "diff", false, "diff")
	cmdFlags.BoolVar(&c.check, "check", false, "check")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The fmt command expects at most one argument.")
		cmdFlags.Usage()
		return 1
	}

	var dirs []string
	if len(args) == 0 {
		dirs = []string{"."}
	} else if args[0] == stdinArg {
		c.opts.List = false
		c.opts.Write = false
	} else {
		dirs = []string{args[0]}
	}

	var output io.Writer
	list := c.opts.List // preserve the original value of -list
	if c.check {
		// set to true so we can use the list output to check
		// if the input needs formatting
		c.opts.List = true
		c.opts.Write = false
		output = &bytes.Buffer{}
	} else {
		output = &cli.UiWriter{Ui: c.Ui}
	}

	err = fmtcmd.Run(dirs, []string{fileExtension}, c.input, output, c.opts)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error running fmt: %s", err))
		return 2
	}

	if c.check {
		buf := output.(*bytes.Buffer)
		ok := buf.Len() == 0
		if list {
			io.Copy(&cli.UiWriter{Ui: c.Ui}, buf)
		}
		if ok {
			return 0
		} else {
			return 3
		}
	}

	return 0
}

func (c *FmtCommand) Help() string {
	helpText := `
Usage: terraform fmt [options] [DIR]

	Rewrites all Terraform configuration files to a canonical format.

	If DIR is not specified then the current working directory will be used.
	If DIR is "-" then content will be read from STDIN.

Options:

  -list=true       List files whose formatting differs (always false if using STDIN)

  -write=true      Write result to source file instead of STDOUT (always false if using STDIN or -check)

  -diff=false      Display diffs of formatting changes

  -check=false     Check if the input is formatted. Exit status will be 0 if all input is properly formatted and non-zero otherwise.

`
	return strings.TrimSpace(helpText)
}

func (c *FmtCommand) Synopsis() string {
	return "Rewrites config files to canonical format"
}
