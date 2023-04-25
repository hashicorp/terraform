package renderers

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/jsondiff"
	"github.com/hashicorp/terraform/internal/plans"
)

// RendererJsonOpts creates a jsondiff.JsonOpts object that returns the correct
// embedded renderers for each JSON type.
//
// We need to define this in our renderers package in order to avoid cycles, and
// to help with reuse between the output processing in the differs package, and
// our JSON string rendering here.
func RendererJsonOpts() jsondiff.JsonOpts {
	return jsondiff.JsonOpts{
		Primitive: func(before, after interface{}, ctype cty.Type, action plans.Action, importing bool) computed.Diff {
			return computed.NewDiff(Primitive(before, after, ctype), action, false, importing)
		},
		Object: func(elements map[string]computed.Diff, action plans.Action, importing bool) computed.Diff {
			return computed.NewDiff(Object(elements), action, false, importing)
		},
		Array: func(elements []computed.Diff, action plans.Action, importing bool) computed.Diff {
			return computed.NewDiff(List(elements), action, false, importing)
		},
		Unknown: func(diff computed.Diff, action plans.Action, importing bool) computed.Diff {
			return computed.NewDiff(Unknown(diff), action, false, importing)
		},
		Sensitive: func(diff computed.Diff, beforeSensitive bool, afterSensitive bool, action plans.Action, importing bool) computed.Diff {
			return computed.NewDiff(Sensitive(diff, beforeSensitive, afterSensitive), action, false, importing)
		},
		TypeChange: func(before, after computed.Diff, action plans.Action, importing bool) computed.Diff {
			return computed.NewDiff(TypeChange(before, after), action, false, importing)
		},
	}
}
