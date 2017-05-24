package atlas

import (
	"github.com/r3labs/terraform/backend"
)

// backend.CLI impl.
func (b *Backend) CLIInit(opts *backend.CLIOpts) error {
	b.CLI = opts.CLI
	b.CLIColor = opts.CLIColor
	b.ContextOpts = opts.ContextOpts
	return nil
}
