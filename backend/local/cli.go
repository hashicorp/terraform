package local

import (
	"github.com/hashicorp/terraform/backend"
)

// backend.CLI impl.
func (b *Local) CLIInit(opts *backend.CLIOpts) error {
	b.CLI = opts.CLI
	b.CLIColor = opts.CLIColor
	b.ShowDiagnostics = opts.ShowDiagnostics
	b.ContextOpts = opts.ContextOpts
	b.OpInput = opts.Input
	b.OpValidation = opts.Validation
	b.RunningInAutomation = opts.RunningInAutomation

	// configure any new cli options
	if opts.StatePath != "" {
		b.StatePath = opts.StatePath
	}

	if opts.StateOutPath != "" {
		b.StateOutPath = opts.StateOutPath
	}

	if opts.StateBackupPath != "" {
		b.StateBackupPath = opts.StateBackupPath
	}

	return nil
}
