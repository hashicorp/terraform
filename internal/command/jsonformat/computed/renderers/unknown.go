// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*unknownRenderer)(nil)

func Unknown(before computed.Diff) computed.DiffRenderer {
	return &unknownRenderer{
		before: before,
	}
}

type unknownRenderer struct {
	NoWarningsRenderer

	before computed.Diff
}

func (renderer unknownRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	if diff.Action == plans.Create {
		return fmt.Sprintf("(known after apply)%s", forcesReplacement(diff.Replace, opts))
	}

	beforeOpts := opts.Clone()
	// Never render null suffix for children of unknown changes.
	beforeOpts.OverrideNullSuffix = true
	if diff.Replace {
		// If we're displaying forces replacement for the overall unknown
		// change, then do not display it for the before specifically.
		beforeOpts.ForbidForcesReplacement = true
	}
	return fmt.Sprintf("%s -> (known after apply)%s", renderer.before.RenderHuman(indent, beforeOpts), forcesReplacement(diff.Replace, opts))
}
