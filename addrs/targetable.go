package addrs

// Targetable is an interface implemented by all address types that can be
// used as "targets" for selecting sub-graphs of a graph.
type Targetable interface {
	targetableSigil()

	// TargetContains returns true if the receiver is considered to contain
	// the given other address. Containment, for the purpose of targeting,
	// means that if a container address is targeted then all of the
	// addresses within it are also implicitly targeted.
	//
	// A targetable address always contains at least itself.
	TargetContains(other Targetable) bool

	// String produces a string representation of the address that could be
	// parsed as a HCL traversal and passed to ParseTarget to produce an
	// identical result.
	String() string
}

type targetable struct {
}

func (r targetable) targetableSigil() {
}
