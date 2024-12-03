// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
)

var _ computed.DiffRenderer = (*writeOnlyRenderer)(nil)

func WriteOnly() computed.DiffRenderer {
	return &writeOnlyRenderer{
		// inner: change,
		// beforeSensitive: beforeSensitive,
		// afterSensitive:  afterSensitive,
	}
}

type writeOnlyRenderer struct {
	// inner computed.Diff

	// beforeSensitive bool
	// afterSensitive  bool
}

func (renderer writeOnlyRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	return fmt.Sprintf("(write-only attribute)%s%s", nullSuffix(diff.Action, opts), forcesReplacement(diff.Replace, opts))
}

func (renderer writeOnlyRenderer) WarningsHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) []string {
	// if (renderer.beforeSensitive == renderer.afterSensitive) || renderer.inner.Action == plans.Create || renderer.inner.Action == plans.Delete {
	// 	// Only display warnings for sensitive values if they are changing from
	// 	// being sensitive or to being sensitive and if they are not being
	// 	// destroyed or created.
	// 	return []string{}
	// }

	// var warning string
	// if renderer.beforeSensitive {
	// 	warning = opts.Colorize.Color(fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will no longer be marked as sensitive\n%s  # after applying this change.", formatIndent(indent)))
	// } else {
	// 	warning = opts.Colorize.Color(fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will be marked as sensitive and will not\n%s  # display in UI output after applying this change.", formatIndent(indent)))
	// }

	// if renderer.inner.Action == plans.NoOp {
	// 	return []string{fmt.Sprintf("%s The value is unchanged.", warning)}
	// }
	return []string{}
}
