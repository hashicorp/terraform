package addrs

// Referenceable is an interface implemented by all address types that can
// appear as references in configuration language expressions.
type Referenceable interface {
	referenceableSigil()
}

type referenceable struct {
}

func (r referenceable) referenceableSigil() {
}
