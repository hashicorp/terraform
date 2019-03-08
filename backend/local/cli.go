package local

import (
	"github.com/hashicorp/terraform/backend"
)

// backend.CLI impl.
func (b *Local) CLIInit(opts *backend.CLIOpts) error {
	if cli, ok := b.Backend.(backend.CLI); ok {
		if err := cli.CLIInit(opts); err != nil {
			return err
		}
	}

	b.CLI = opts.CLI
	b.CLIColor = opts.CLIColor
	b.ContextOpts = opts.ContextOpts
	b.OpInput = opts.Input
	b.OpValidation = opts.Validation
	b.RunningInAutomation = opts.RunningInAutomation

	// Only configure state paths if we didn't do so via the configure func.
	if b.StatePath == "" {
		b.StatePath = opts.StatePath
		b.StateOutPath = opts.StateOutPath
		b.StateBackupPath = opts.StateBackupPath
	}

	return nil
}
