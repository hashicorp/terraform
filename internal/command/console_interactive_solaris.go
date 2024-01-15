// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build solaris
// +build solaris

package command

import (
	"fmt"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/repl"
)

func (c *ConsoleCommand) modeInteractive(session *repl.Session, ui cli.Ui) int {
	ui.Error(fmt.Sprintf(
		"The readline library Terraform currently uses for the interactive\n" +
			"console is not supported by Solaris. Interactive mode is therefore\n" +
			"not supported on Solaris currently."))
	return 1
}
