// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
)

var _ computed.DiffRenderer = (*writeOnlyRenderer)(nil)

func WriteOnly(sensitive bool) computed.DiffRenderer {
	return &writeOnlyRenderer{
		sensitive,
	}
}

type writeOnlyRenderer struct {
	sensitive bool
}

func (renderer writeOnlyRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	if renderer.sensitive {
		return fmt.Sprintf("(sensitive, write-only attribute)%s%s", nullSuffix(diff.Action, opts), forcesReplacement(diff.Replace, opts))
	}
	return fmt.Sprintf("(write-only attribute)%s%s", nullSuffix(diff.Action, opts), forcesReplacement(diff.Replace, opts))
}

func (renderer writeOnlyRenderer) WarningsHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) []string {
	return []string{}
}
