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
	UniqueKeyer

	checkableSigil()

	// Check returns the address of an individual check rule of a specified
	// type and index within this checkable container.
	Check(CheckType, int) Check

	// ConfigCheckable returns the address of the configuration construct that
	// this Checkable belongs to.
	//
	// Checkable objects can potentially be dynamically declared during a
	// plan operation using constructs like resource for_each, and so
	// ConfigCheckable gives us a way to talk about the static containers
	// those dynamic objects belong to, in case we wish to group together
	// dynamic checkable objects into their static checkable for reporting
	// purposes.
	ConfigCheckable() ConfigCheckable

	String() string
}

var (
	_ Checkable = AbsResourceInstance{}
	_ Checkable = AbsOutputValue{}
)

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

// ConfigCheckable is an interfaces implemented by address types that represent
// configuration constructs that can have Checkable addresses associated with
// them.
//
// This address type therefore in a sense represents a container for zero or
// more checkable objects all declared by the same configuration construct,
// so that we can talk about these groups of checkable objects before we're
// ready to decide how many checkable objects belong to each one.
type ConfigCheckable interface {
	UniqueKeyer

	configCheckableSigil()

	String() string
}

var (
	_ ConfigCheckable = ConfigResource{}
	_ ConfigCheckable = ConfigOutputValue{}
)
