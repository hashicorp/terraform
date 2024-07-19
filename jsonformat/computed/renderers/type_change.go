// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"fmt"

	"github.com/hashicorp/terraform/jsonformat/computed"
)

var _ computed.DiffRenderer = (*typeChangeRenderer)(nil)

func TypeChange(before, after computed.Diff) computed.DiffRenderer {
	return &typeChangeRenderer{
		before: before,
		after:  after,
	}
}

type typeChangeRenderer struct {
	NoWarningsRenderer

	before computed.Diff
	after  computed.Diff
}

func (renderer typeChangeRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	opts.OverrideNullSuffix = true // Never render null suffix for children of type changes.
	return fmt.Sprintf("%s %s %s", renderer.before.RenderHuman(indent, opts), opts.Colorize.Color("[yellow]->[reset]"), renderer.after.RenderHuman(indent, opts))
}
