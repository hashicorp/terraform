package change

import (
	"fmt"
	"reflect"
)

func Sensitive(before, after interface{}, beforeSensitive, afterSensitive bool) Renderer {
	return &sensitiveRenderer{
		before:          before,
		after:           after,
		beforeSensitive: beforeSensitive,
		afterSensitive:  afterSensitive,
	}
}

type sensitiveRenderer struct {
	before interface{}
	after  interface{}

	beforeSensitive bool
	afterSensitive  bool
}

func (renderer sensitiveRenderer) Render(change Change, indent int, opts RenderOpts) string {
	return fmt.Sprintf("(sensitive)%s%s", change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
}

func (renderer sensitiveRenderer) Warnings(change Change, indent int) []string {
	if (renderer.beforeSensitive == renderer.afterSensitive) || renderer.before == nil || renderer.after == nil {
		// Only display warnings for sensitive values if they are changing from
		// being sensitive or to being sensitive or if they are not being
		// destroyed or created.
		return []string{}
	}

	var warning string
	if renderer.beforeSensitive {
		warning = fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will no longer be marked as sensitive\n%s  # after applying this change.", change.indent(indent))
	} else {
		warning = fmt.Sprintf("  # [yellow]Warning[reset]: this attribute value will be marked as sensitive and will not\n%s  # display in UI output after applying this change.", change.indent(indent))
	}

	if reflect.DeepEqual(renderer.before, renderer.after) {
		return []string{fmt.Sprintf("%s The value is unchanged.", warning)}
	}
	return []string{warning}
}
