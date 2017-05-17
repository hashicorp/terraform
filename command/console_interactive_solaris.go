// +build solaris

package command

import (
	"fmt"

	"github.com/hashicorp/terraform/repl"
	"github.com/mitchellh/cli"
)

func (c *ConsoleCommand) modeInteractive(session *repl.Session, ui cli.Ui) int {
	ui.Error(fmt.Sprintf(
		"The readline library Terraform currently uses for the interactive\n" +
			"console is not supported by Solaris. Interactive mode is therefore\n" +
			"not supported on Solaris currently."))
	return 1
}
