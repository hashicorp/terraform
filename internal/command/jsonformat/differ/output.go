package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
)

func (change Change) ComputeDiffForOutput() computed.Diff {
	if sensitive, ok := change.checkForSensitiveType(cty.DynamicPseudoType); ok {
		return sensitive
	}

	if unknown, ok := change.checkForUnknownType(cty.DynamicPseudoType); ok {
		return unknown
	}

	jsonOpts := renderers.RendererJsonOpts()
	return jsonOpts.Transform(change.Before, change.After, change.BeforeExplicit, change.AfterExplicit, change.RelevantAttributes)
}
