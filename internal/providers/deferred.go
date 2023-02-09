package providers

import (
	"github.com/zclconf/go-cty/cty"
)

// DeferredReason describes one reason why a particular action cannot be
// completed fully in the current run, and must therefore be deferred to
// a later run where hopefully more informationis available.
//
// This is a sealed interface type whose full set of implementations lives
// inside this package. Future versions of this package may include additional
// implementations, so callers should use [DeferredReasonOther] as a placeholder
// if they encounter a [DeferredReason] type that they don't recognize.
type DeferredReason interface {
	deferredReasonSigil()
}

func NewDeferredReasonOther() DeferredReason {
	return DeferredReasonOther{}
}

func NewDeferredReasonUnknownProviderConfig(attributePath cty.Path) DeferredReason {
	return DeferredReasonUnknownProviderConfig{
		AttributePath: attributePath,
	}
}

func NewDeferredReasonUnknownResourceConfig(attributePath cty.Path) DeferredReason {
	return DeferredReasonUnknownResourceConfig{
		AttributePath: attributePath,
	}
}

// DeferredReasonOther is the [DeferredReason] implementation to use when
// none of the others are applicable, and also a good placeholder to use if
// a caller encounters a reason type it doesn't recognize.
type DeferredReasonOther struct{}

func (DeferredReasonOther) deferredReasonSigil() {}

// DeferredReasonUnknownProviderConfig is the [DeferredReason] implementation
// to represent that an unknown attribute value in the configuration of the
// provider responsible for the action prevents full planning or execution
// of the action.
type DeferredReasonUnknownProviderConfig struct {
	// AttributePath is a path to the closest possible attribute to the one
	// whose unknown value caused the problem, which Terraform might then use
	// to highlight the particular attribute somehow in the UI.
	// This path is resolved within the provider configuration, not within
	// the configuration of a resource belonging to the provider.
	AttributePath cty.Path
}

func (DeferredReasonUnknownProviderConfig) deferredReasonSigil() {}

// DeferredReasonUnknownProviderConfig is the [DeferredReason] implementation
// to represent that an unknown attribute value in the configuration of the
// resource an action relates to prevents full planning or execution of the
// action.
type DeferredReasonUnknownResourceConfig struct {
	// AttributePath is a path to the closest possible attribute to the one
	// whose unknown value caused the problem, which Terraform might then use
	// to highlight the particular attribute somehow in the UI.
	AttributePath cty.Path
}

func (DeferredReasonUnknownResourceConfig) deferredReasonSigil() {}
