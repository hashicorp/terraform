package remote

import (
	"github.com/hashicorp/terraform/backend"
)

// CLIInit implements backend.CLI
func (b *Remote) CLIInit(opts *backend.CLIOpts) error {
	b.CLI = opts.CLI
	b.CLIColor = opts.CLIColor
	b.ShowDiagnostics = opts.ShowDiagnostics
	b.ContextOpts = opts.ContextOpts
	return nil
}
