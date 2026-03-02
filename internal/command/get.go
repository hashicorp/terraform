// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GetCommand is a Command implementation that takes a Terraform
// configuration and downloads all the modules.
type GetCommand struct {
	Meta
}

func (c *GetCommand) Run(args []string) int {
	args = c.Meta.process(args)
	parsedArgs, diags := arguments.ParseGet(args)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Initialization can be aborted by interruption signals
	ctx, done := c.InterruptibleContext(c.CommandContext())
	defer done()

	path, err := ModulePath(parsedArgs.Args)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	path = c.normalizePath(path)

	abort, diags := getModules(ctx, &c.Meta, path, parsedArgs.TestsDirectory, parsedArgs.Update)
	c.showDiagnostics(diags)
	if abort || diags.HasErrors() {
		return 1
	}

	return 0
}

func (c *GetCommand) Help() string {
	helpText := `
Usage: terraform [global options] get [options]

  Downloads and installs modules needed for the configuration in the 
  current working directory.

  This recursively downloads all modules needed, such as modules
  imported by modules imported by the root and so on. If a module is
  already downloaded, it will not be redownloaded or checked for updates
  unless the -update flag is specified.

  Module installation also happens automatically by default as part of
  the "terraform init" command, so you should rarely need to run this
  command separately.

Options:

  -update               Check already-downloaded modules for available updates
                        and install the newest versions available.

  -no-color             Disable text coloring in the output.

  -test-directory=path	Set the Terraform test directory, defaults to "tests".

`
	return strings.TrimSpace(helpText)
}

func (c *GetCommand) Synopsis() string {
	return "Install or upgrade remote Terraform modules"
}

func getModules(ctx context.Context, m *Meta, path string, testsDir string, upgrade bool) (abort bool, diags tfdiags.Diagnostics) {
	hooks := uiModuleInstallHooks{
		Ui:             m.Ui,
		ShowLocalPaths: true,
	}
	return m.installModules(ctx, path, testsDir, upgrade, true, hooks)
}
