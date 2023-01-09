package change

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
)

func Sensitive(change Change, beforeSensitive, afterSensitive bool) Renderer {
	return &sensitiveRenderer{
		change:          change,
		beforeSensitive: beforeSensitive,
		afterSensitive:  afterSensitive,
	}
}

type sensitiveRenderer struct {
	change Change

	beforeSensitive bool
	afterSensitive  bool
}

func (renderer sensitiveRenderer) Render(change Change, indent int, opts RenderOpts) string {
	return fmt.Sprintf("(sensitive)%s%s", change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
}

func (renderer sensitiveRenderer) Warnings(change Change, indent int) []string {
	if (renderer.beforeSensitive == renderer.afterSensitive) || renderer.change.action == plans.Create || renderer.change.action == plans.Delete {
		// Only display warnings for sensitive values if they are changing from
		// being sensitive or to being sensitive and if they are not being
		// destroyed or created.
		return []string{}
	}

	var warning string
	if renderer.beforeSensitive {
		warning = fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will no longer be marked as sensitive\n%s  # after applying this change.", change.indent(indent))
	} else {
		warning = fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will be marked as sensitive and will not\n%s  # display in UI output after applying this change.", change.indent(indent))
	}

	if renderer.change.action == plans.NoOp {
		return []string{fmt.Sprintf("%s The value is unchanged.", warning)}
	}
	return []string{warning}
}
