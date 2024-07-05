package addrs

// CustomActionStep represents a reference to an earlier step in a sequence
// of custom action invocations.
type CustomActionStep struct {
	Name string
}

var _ Referenceable = CustomActionStep{}
var _ UniqueKey = CustomActionStep{}

// String implements Referenceable.
func (c CustomActionStep) String() string {
	return "step." + c.Name
}

// UniqueKey implements Referenceable.
func (c CustomActionStep) UniqueKey() UniqueKey {
	return c
}

// referenceableSigil implements Referenceable.
func (c CustomActionStep) referenceableSigil() {}

// uniqueKeySigil implements UniqueKey.
func (c CustomActionStep) uniqueKeySigil() {}
