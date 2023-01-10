package differ

import (
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"

	"github.com/zclconf/go-cty/cty"
)

func (change Change) ComputeDiffForOutput() computed.Diff {
	if sensitive, ok := change.checkForSensitiveType(cty.DynamicPseudoType); ok {
		return sensitive
	}

	if unknown, ok := change.checkForUnknownType(cty.DynamicPseudoType); ok {
		return unknown
	}

	jsonOpts := renderers.DefaultJsonOpts()
	return jsonOpts.Transform(change.Before, change.After)
}
