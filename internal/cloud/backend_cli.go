// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/views"
)

// CLIInit implements backendrun.CLI
func (b *Cloud) CLIInit(opts *backendrun.CLIOpts) error {
	if cli, ok := b.local.(backendrun.CLI); ok {
		if err := cli.CLIInit(opts); err != nil {
			return err
		}
	}

	b.CLI = opts.CLI
	b.CLIColor = opts.CLIColor
	b.ContextOpts = opts.ContextOpts
	b.runningInAutomation = opts.RunningInAutomation
	b.input = opts.Input
	b.renderer = &jsonformat.Renderer{
		Streams:  opts.Streams,
		Colorize: opts.CLIColor,
	}
	b.View = views.NewCloud(opts.ViewType, opts.View)

	return nil
}
