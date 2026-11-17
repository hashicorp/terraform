// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"log"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
)

// backendrun.CLI impl.
func (b *Local) CLIInit(opts *backendrun.CLIOpts) error {
	b.ContextOpts = opts.ContextOpts
	b.OpInput = opts.Input
	b.OpValidation = opts.Validation

	// configure any new cli options
	if opts.OverrideStatePath != "" {
		log.Printf("[TRACE] backend/local: CLI option -state is overriding state path to %s", opts.OverrideStatePath)
		b.overrideStatePath = opts.OverrideStatePath
	}

	if opts.OverrideStateOutPath != "" {
		log.Printf("[TRACE] backend/local: CLI option -state-out is overriding state output path to %s", opts.OverrideStateOutPath)
		b.overrideStateOutPath = opts.OverrideStateOutPath
	}

	if opts.OverrideStateBackupPath != "" {
		log.Printf("[TRACE] backend/local: CLI option -backup is overriding state backup path to %s", opts.OverrideStateBackupPath)
		b.overrideStateBackupPath = opts.OverrideStateBackupPath
	}

	return nil
}
