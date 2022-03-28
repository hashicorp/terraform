package addrs

import "fmt"

// Check is the address of a check rule within a checkable object.
//
// This represents the check rule globally within a configuration, and is used
// during graph evaluation to identify a condition result object to update with
// the result of check rule evaluation.
//
// The check address is not distinct from resource traversals, and check rule
// values are not intended to be available to the language, so the address is
// not Referenceable.
//
// Note also that the check address is only relevant within the scope of a run,
// as reordering check blocks between runs will result in their addresses
// changing.
type Check struct {
	Container Checkable
	Type      CheckType
	Index     int
}

func (c Check) String() string {
	container := c.Container.String()
	switch c.Type {
	case ResourcePrecondition:
		return fmt.Sprintf("%s.preconditions[%d]", container, c.Index)
	case ResourcePostcondition:
		return fmt.Sprintf("%s.postconditions[%d]", container, c.Index)
	case OutputPrecondition:
		return fmt.Sprintf("%s.preconditions[%d]", container, c.Index)
	default:
		// This should not happen
		return fmt.Sprintf("%s.conditions[%d]", container, c.Index)
	}
}

// Checkable is an interface implemented by all address types that can contain
// condition blocks.
type Checkable interface {
	checkableSigil()

	// Check returns the address of an individual check rule of a specified
	// type and index within this checkable container.
	Check(CheckType, int) Check
	String() string
}

var (
	_ Checkable = AbsResourceInstance{}
	_ Checkable = AbsOutputValue{}
)

type checkable struct {
}

func (c checkable) checkableSigil() {
}

// CheckType describes the category of check.
//go:generate go run golang.org/x/tools/cmd/stringer -type=CheckType check.go
type CheckType int

const (
	InvalidCondition      CheckType = 0
	ResourcePrecondition  CheckType = 1
	ResourcePostcondition CheckType = 2
	OutputPrecondition    CheckType = 3
)

// Description returns a human-readable description of the check type. This is
// presented in the user interface through a diagnostic summary.
func (c CheckType) Description() string {
	switch c {
	case ResourcePrecondition:
		return "Resource precondition"
	case ResourcePostcondition:
		return "Resource postcondition"
	case OutputPrecondition:
		return "Module output value precondition"
	default:
		// This should not happen
		return "Condition"
	}
}
