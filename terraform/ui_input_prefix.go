package terraform

import (
	"fmt"
)

// PrefixUIInput is an implementation of UIInput that prefixes the ID
// with a string, allowing queries to be namespaced.
type PrefixUIInput struct {
	IdPrefix string
	UIInput  UIInput
}

func (i *PrefixUIInput) Input(opts *InputOpts) (string, error) {
	opts.Id = fmt.Sprintf("%s.%s", i.IdPrefix, opts.Id)
	return i.UIInput.Input(opts)
}
