package local

import (
	"log"

	"github.com/hashicorp/terraform/backend"
)

// backend.CLI impl.
func (b *Local) CLIInit(opts *backend.CLIOpts) error {
	b.ContextOpts = opts.ContextOpts
	b.OpInput = opts.Input
	b.OpValidation = opts.Validation

	// configure any new cli options
	if opts.StatePath != "" {
		log.Printf("[TRACE] backend/local: CLI option -state is overriding state path to %s", opts.StatePath)
		b.OverrideStatePath = opts.StatePath
	}

	if opts.StateOutPath != "" {
		log.Printf("[TRACE] backend/local: CLI option -state-out is overriding state output path to %s", opts.StateOutPath)
		b.OverrideStateOutPath = opts.StateOutPath
	}

	if opts.StateBackupPath != "" {
		log.Printf("[TRACE] backend/local: CLI option -backup is overriding state backup path to %s", opts.StateBackupPath)
		b.OverrideStateBackupPath = opts.StateBackupPath
	}

	return nil
}
