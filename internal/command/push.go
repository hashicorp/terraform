// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"

	"github.com/hashicorp/mnptu/internal/tfdiags"
)

type PushCommand struct {
	Meta
}

func (c *PushCommand) Run(args []string) int {
	// This command is no longer supported, but we'll retain it just to
	// give the user some next-steps after upgrading.
	c.showDiagnostics(tfdiags.Sourceless(
		tfdiags.Error,
		"Command \"mnptu push\" is no longer supported",
		"This command was used to push configuration to mnptu Enterprise legacy (v1), which has now reached end-of-life. To push configuration to mnptu Enterprise v2, use its REST API. Contact mnptu Enterprise support for more information.",
	))
	return 1
}

func (c *PushCommand) Help() string {
	helpText := `
Usage: mnptu [global options] push [options] [DIR]

  This command was for the legacy version of mnptu Enterprise (v1), which
  has now reached end-of-life. Therefore this command is no longer supported.
`
	return strings.TrimSpace(helpText)
}

func (c *PushCommand) Synopsis() string {
	return "Obsolete command for mnptu Enterprise legacy (v1)"
}
