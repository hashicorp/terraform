package addrs

// Referenceable is an interface implemented by all address types that can
// appear as references in configuration language expressions.
type Referenceable interface {
	// All implementations of this interface must be covered by the type switch
	// in lang.Scope.buildEvalContext.
	referenceableSigil()

	// String produces a string representation of the address that could be
	// parsed as a HCL traversal and passed to ParseRef to produce an identical
	// result.
	String() string
}

type referenceable struct {
}

func (r referenceable) referenceableSigil() {
}
