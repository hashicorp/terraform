// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
)

// TestCleanupCommand is a command that cleans up left-over resources created
// during Terraform test runs. It basically runs the test command in cleanup mode.
type TestCleanupCommand struct {
	*TestCommand
}

func (c *TestCleanupCommand) Help() string {
	helpText := `
Usage: terraform [global options] test cleanup [options]

  Cleans up left-over resources created during Terraform test runs.

  By default, this command ignores the skip_cleanup attributes in the manifest
  file. Use the -repair flag to override this behavior. Additionally, the
  -target flag allows specifying which state files to clean up.

Options:

  -repair               Overrides the skip_cleanup attribute in the manifest
                        file and attempts to clean up all resources.

  -target=statefile     Specifies the state file(s) to clean up. Use this option
                        multiple times to target multiple state files.

  -no-color             If specified, output won't contain any color.

  -verbose              Print detailed output during the cleanup process.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCleanupCommand) Synopsis() string {
	return "Clean up left-over resources created during Terraform test runs"
}

func (c *TestCleanupCommand) Run(rawArgs []string) int {
	c.TestCommand.cleanupMode = true
	return c.TestCommand.Run(rawArgs)
}
