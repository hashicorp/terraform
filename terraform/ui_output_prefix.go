package terraform

import (
	"fmt"
)

// PrefixUIOutput is an implementation of UIOutput that prefixes the output
// with a string.
type PrefixUIOutput struct {
	Prefix string
	UIOutput     UIOutput
}

func (i *PrefixUIOutput) Output(v string) {
	v = fmt.Sprintf("%s%s", i.Prefix, v)
	i.UIOutput.Output(v)
}
