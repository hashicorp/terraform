package command

import (
	"sync"

	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// UIOutput is an implementation of terraform.UIOutput.
type UIOutput struct {
	Colorize *colorstring.Colorize
	Ui       cli.Ui

	once sync.Once
	ui        cli.Ui
}

func (u *UIOutput) Output(v string) {
	u.once.Do(u.init)
	u.ui.Output(v)
}

func (u *UIOutput) init() {
	// Wrap the ui so that it is safe for concurrency regardless of the
	// underlying reader/writer that is in place.
	u.ui = &cli.ConcurrentUi{Ui: u.Ui}
}
