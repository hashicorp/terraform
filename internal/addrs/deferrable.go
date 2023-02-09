package addrs

import (
	"strings"
)

// Deferable represents addresses of types of objects that can have actions
// that might be deferred for execution in a later run.
//
// When an action is defered we still need to describe an approximation of
// the effects of that action so that users can get early feedback about
// whether applying the non-deferred changes will move them closer to their
// desired state. Deferable addresses are how we communicate which object
// each deferred action relates to.
type Deferrable interface {
	deferrableSigil()
	UniqueKeyer

	// DeferrableString is like String but returns an address form specifically
	// tailored for a UI that is describing deferred changes, which clearly
	// distinguishes between the different possible deferable address types.
	DeferrableString() string
}

// ConfigResource is deferable because sometimes we must defer even the
// expansion of a resource due to either its own repetition argument or that
// of one of its containing modules being unknown.
func (ConfigResource) deferrableSigil() {}

func (r ConfigResource) DeferrableString() string {
	// Because deferred unexpanded resources will be shown in the same context
	// as expanded resource instances, we'll use a special format here to
	// make it explicit that we're talking about all instances of a particular
	// resource or module, rather than the _unkeyed instance_ of each.
	// This follows a similar convention to how we display "move endpoints"
	// in the UI, like [MoveEndpointInModule.String]:
	// module.foo[*].aws_instance.bar[*], to differentiate from
	// module.foo.aws_instance.bar the single instance.
	var buf strings.Builder
	for _, name := range r.Module {
		buf.WriteString("module.")
		buf.WriteString(name)
		buf.WriteString("[*].")
	}
	buf.WriteString(r.Resource.String())
	buf.WriteString("[*]")
	return buf.String()
}

var _ Deferrable = ConfigResource{}

// AbsResourceInstance is deferable for situations where we have already
// succeeded in expanding a resource but one of its instances must be
// defered either for provider-specific reasons or because it is downstream
// of some other deferred action.
func (AbsResourceInstance) deferrableSigil() {}

var _ Deferrable = AbsResourceInstance{}

func (r AbsResourceInstance) DeferrableString() string {
	// Our "deferrable" string format for AbsResourceInstance is just its
	// normal string format, because this is the main case.
	return r.String()
}
