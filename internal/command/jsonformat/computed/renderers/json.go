package renderers

import (
	"fmt"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/zclconf/go-cty/cty"
	"strings"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/jsondiff"
	"github.com/hashicorp/terraform/internal/plans"
)

func DefaultJsonOpts() jsondiff.JsonOpts {
	return jsondiff.JsonOpts{
		Primitive: func(before, after interface{}, ctype cty.Type, action plans.Action) computed.Diff {
			return computed.NewDiff(Primitive(before, after, ctype), action, false)
		},
		Object: func(elements map[string]computed.Diff, action plans.Action) computed.Diff {
			return computed.NewDiff(Object(elements), action, false)
		},
		Array: func(elements []computed.Diff, action plans.Action) computed.Diff {
			return computed.NewDiff(List(elements), action, false)
		},
		TypeChange: func(before, after computed.Diff, action plans.Action) computed.Diff {
			return computed.NewDiff(TypeChange(before, after), action, false)
		},
	}
}

func (renderer primitiveRenderer) renderStringDiffAsJson(diff computed.Diff, indent int, opts computed.RenderHumanOpts, before evaluatedString, after evaluatedString) string {
	jsonDiff := DefaultJsonOpts().Transform(before.Json, after.Json)

	var whitespace, replace string
	if jsonDiff.Action == plans.NoOp && diff.Action == plans.Update {
		// Then this means we are rendering a whitespace only change. The JSON
		// differ will have ignored the whitespace changes so that makes the
		// diff we are about to print out very confusing.
		if diff.Replace {
			whitespace = " # whitespace changes force replacement"
		} else {
			whitespace = " # whitespace changes"
		}
	} else {
		// We only show the replace suffix if we didn't print something out
		// about whitespace changes.
		replace = forcesReplacement(diff.Replace)
	}

	renderedJsonDiff := jsonDiff.RenderHuman(indent, opts)

	if strings.Contains(renderedJsonDiff, "\n") {
		return fmt.Sprintf("jsonencode(%s\n%s%s %s%s\n%s)", whitespace, formatIndent(indent), format.DiffActionSymbol(diff.Action), renderedJsonDiff, replace, formatIndent(indent))
	}
	return fmt.Sprintf("jsonencode(%s)%s%s", renderedJsonDiff, whitespace, replace)
}
