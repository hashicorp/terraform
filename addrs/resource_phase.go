package addrs

import "fmt"

// ResourceInstancePhase is a special kind of reference used only internally
// during graph building to represent resource instances that are in a
// non-primary state.
//
// Graph nodes can declare themselves referenceable via an instance phase
// or can declare that they reference an instance phase in order to accomodate
// secondary graph nodes dealing with, for example, destroy actions.
//
// This special reference type cannot be accessed directly by end-users, and
// should never be shown in the UI.
type ResourceInstancePhase struct {
	referenceable
	ResourceInstance ResourceInstance
	Phase            ResourceInstancePhaseType
}

var _ Referenceable = ResourceInstancePhase{}

// Phase returns a special "phase address" for the receving instance. See the
// documentation of ResourceInstancePhase for the limited situations where this
// is intended to be used.
func (r ResourceInstance) Phase(rpt ResourceInstancePhaseType) ResourceInstancePhase {
	return ResourceInstancePhase{
		ResourceInstance: r,
		Phase:            rpt,
	}
}

func (rp ResourceInstancePhase) String() string {
	// We use a different separator here than usual to ensure that we'll
	// never conflict with any non-phased resource instance string. This
	// is intentionally something that would fail parsing with ParseRef,
	// because this special address type should never be exposed in the UI.
	return fmt.Sprintf("%s#%s", rp.ResourceInstance, rp.Phase)
}

// ResourceInstancePhaseType is an enumeration used with ResourceInstancePhase.
type ResourceInstancePhaseType string

const (
	// ResourceInstancePhaseDestroy represents the "destroy" phase of a
	// resource instance.
	ResourceInstancePhaseDestroy ResourceInstancePhaseType = "destroy"

	// ResourceInstancePhaseDestroyCBD is similar to ResourceInstancePhaseDestroy
	// but is used for resources that have "create_before_destroy" set, thus
	// requiring a different dependency ordering.
	ResourceInstancePhaseDestroyCBD ResourceInstancePhaseType = "destroy-cbd"
)

func (rpt ResourceInstancePhaseType) String() string {
	return string(rpt)
}
