package command

import (
	"github.com/mitchellh/colorstring"
)

// Meta are the meta-options that are available on all or most commands.
type Meta struct {
	Color bool
}

// Colorize returns the colorization structure for a command.
func (m *Meta) Colorize() *colorstring.Colorize {
	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: !m.Color,
		Reset:   true,
	}
}

// process will process the meta-parameters out of the arguments. This
// will potentially modify the args in-place. It will return the resulting
// slice.
func (m *Meta) process(args []string) []string {
	m.Color = true

	for i, v := range args {
		if v == "-no-color" {
			m.Color = false
			return append(args[:i], args[i+1:]...)
		}
	}

	return args
}
