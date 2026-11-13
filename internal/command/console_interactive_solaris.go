// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build solaris
// +build solaris

package command

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/repl"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (c *ConsoleCommand) modeInteractive(session *repl.Session, view views.Console) int {
	view.Diagnostics(tfdiags.Diagnostics{}.Append(fmt.Errorf(
		"The readline library Terraform currently uses for the interactive\n" +
			"console is not supported by Solaris. Interactive mode is therefore\n" +
			"not supported on Solaris currently.")))
	return 1
}
