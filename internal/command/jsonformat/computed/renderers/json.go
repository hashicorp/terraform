// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
		Primitive: func(before, after interface{}, ctype cty.Type, action plans.Action) computed.Diff {
			return computed.NewDiff(Primitive(before, after, ctype), action, false)
		},
		Object: func(elements map[string]computed.Diff, action plans.Action) computed.Diff {
			return computed.NewDiff(Object(elements), action, false)
		},
		Array: func(elements []computed.Diff, action plans.Action) computed.Diff {
			return computed.NewDiff(List(elements), action, false)
		},
		Unknown: func(diff computed.Diff, action plans.Action) computed.Diff {
			return computed.NewDiff(Unknown(diff), action, false)
		},
		Sensitive: func(diff computed.Diff, beforeSensitive bool, afterSensitive bool, action plans.Action) computed.Diff {
			return computed.NewDiff(Sensitive(diff, beforeSensitive, afterSensitive), action, false)
		},
		TypeChange: func(before, after computed.Diff, action plans.Action) computed.Diff {
			return computed.NewDiff(TypeChange(before, after), action, false)
		},
	}
}
