package command

import (
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// UIOutput is an implementation of terraform.UIOutput.
type UIOutput struct {
	Colorize *colorstring.Colorize
	Ui       cli.Ui
}

func (u *UIOutput) Output(v string) {
}
