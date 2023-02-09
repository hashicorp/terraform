package convert

import (
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfplugin6"
)

func DeferredReasonFromProto(protoReason *tfplugin6.DeferredAction) providers.DeferredReason {
	if protoReason == nil {
		// Should never happen, but we'll treat it as an "other" reason
		// for robustness against oddly-behaving providers.
		return providers.NewDeferredReasonOther()
	}

	switch protoReason := protoReason.Reason.(type) {
	case *tfplugin6.DeferredAction_ProviderConfigUnknown:
		return providers.NewDeferredReasonUnknownProviderConfig(
			AttributePathToPath(protoReason.ProviderConfigUnknown.Attribute),
		)
	case *tfplugin6.DeferredAction_ResourceConfigUnknown:
		return providers.NewDeferredReasonUnknownResourceConfig(
			AttributePathToPath(protoReason.ResourceConfigUnknown.Attribute),
		)
	default:
		// Fallback for all unrecognized reasons, in case later protocol
		// versions define additional ones.
		return providers.NewDeferredReasonOther()
	}
}
