// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"log"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	localState "github.com/hashicorp/terraform/internal/backend/local-state"
)

// backendrun.CLI impl.
func (b *Local) CLIInit(opts *backendrun.CLIOpts) error {
	b.ContextOpts = opts.ContextOpts
	b.OpInput = opts.Input
	b.OpValidation = opts.Validation

	// If CLI options affect how local state is stored, set on
	// the internal Backend
	if be, ok := b.Backend.(*localState.Local); ok {
		if opts.StatePath != "" {
			log.Printf("[TRACE] backend/local: CLI option -state is overriding state path to %s", opts.StatePath)
			be.OverrideStatePath = opts.StatePath
		}

		if opts.StateOutPath != "" {
			log.Printf("[TRACE] backend/local: CLI option -state-out is overriding state output path to %s", opts.StateOutPath)
			be.OverrideStateOutPath = opts.StateOutPath
		}

		if opts.StateBackupPath != "" {
			log.Printf("[TRACE] backend/local: CLI option -backup is overriding state backup path to %s", opts.StateBackupPath)
			be.OverrideStateBackupPath = opts.StateBackupPath
		}
	}

	return nil
}
