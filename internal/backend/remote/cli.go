// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"github.com/hashicorp/terraform/internal/backend/backendrun"
)

// CLIInit implements backendrun.CLI
func (b *Remote) CLIInit(opts *backendrun.CLIOpts) error {
	if cli, ok := b.local.(backendrun.CLI); ok {
		if err := cli.CLIInit(opts); err != nil {
			return err
		}
	}

	b.CLI = opts.CLI
	b.CLIColor = opts.CLIColor
	b.ContextOpts = opts.ContextOpts

	return nil
}
