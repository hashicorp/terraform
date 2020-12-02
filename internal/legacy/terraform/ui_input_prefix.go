package terraform

import (
	"context"
	"fmt"
)

// PrefixUIInput is an implementation of UIInput that prefixes the ID
// with a string, allowing queries to be namespaced.
type PrefixUIInput struct {
	IdPrefix    string
	QueryPrefix string
	UIInput     UIInput
}

func (i *PrefixUIInput) Input(ctx context.Context, opts *InputOpts) (string, error) {
	opts.Id = fmt.Sprintf("%s.%s", i.IdPrefix, opts.Id)
	opts.Query = fmt.Sprintf("%s%s", i.QueryPrefix, opts.Query)
	return i.UIInput.Input(ctx, opts)
}
