package renderers

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*sensitiveRenderer)(nil)

func Sensitive(change computed.Diff, beforeSensitive, afterSensitive bool) computed.DiffRenderer {
	return &sensitiveRenderer{
		inner:           change,
		beforeSensitive: beforeSensitive,
		afterSensitive:  afterSensitive,
	}
}

type sensitiveRenderer struct {
	inner computed.Diff

	beforeSensitive bool
	afterSensitive  bool
}

func (renderer sensitiveRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	return fmt.Sprintf("(sensitive)%s%s", nullSuffix(opts.OverrideNullSuffix, diff.Action), forcesReplacement(diff.Replace))
}

func (renderer sensitiveRenderer) WarningsHuman(diff computed.Diff, indent int) []string {
	if (renderer.beforeSensitive == renderer.afterSensitive) || renderer.inner.Action == plans.Create || renderer.inner.Action == plans.Delete {
		// Only display warnings for sensitive values if they are changing from
		// being sensitive or to being sensitive and if they are not being
		// destroyed or created.
		return []string{}
	}

	var warning string
	if renderer.beforeSensitive {
		warning = fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will no longer be marked as sensitive\n%s  # after applying this change.", formatIndent(indent))
	} else {
		warning = fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will be marked as sensitive and will not\n%s  # display in UI output after applying this change.", formatIndent(indent))
	}

	if renderer.inner.Action == plans.NoOp {
		return []string{fmt.Sprintf("%s The value is unchanged.", warning)}
	}
	return []string{warning}
}
