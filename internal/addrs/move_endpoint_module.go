package addrs

import (
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MoveEndpointInModule annotates a MoveEndpoint with the address of the
// module where it was declared, which is the form we use for resolving
// whether move statements chain from or are nested within other move
// statements.
type MoveEndpointInModule struct {
	// SourceRange is the location of the physical endpoint address
	// in configuration, if this MoveEndpoint was decoded from a
	// configuration expresson.
	SourceRange tfdiags.SourceRange

	// The internals are unexported here because, as with MoveEndpoint,
	// we're somewhat abusing AbsMoveable here to represent an address
	// relative to the module, rather than as an absolute address.
	// Conceptually, the following two fields represent a matching pattern
	// for AbsMoveables where the elements of "module" behave as
	// ModuleInstanceStep values with a wildcard instance key, because
	// a moved block in a module affects all instances of that module.
	// Unlike MoveEndpoint, relSubject in this case can be any of the
	// address types that implement AbsMoveable.
	module     Module
	relSubject AbsMoveable
}

func (e *MoveEndpointInModule) ObjectKind() MoveEndpointKind {
	return absMoveableEndpointKind(e.relSubject)
}

// String produces a string representation of the object matching pattern
// represented by the reciever.
//
// Since there is no direct syntax for representing such an object matching
// pattern, this function uses a splat-operator-like representation to stand
// in for the wildcard instance keys.
func (e *MoveEndpointInModule) String() string {
	if e == nil {
		return ""
	}
	var buf strings.Builder
	for _, name := range e.module {
		buf.WriteString("module.")
		buf.WriteString(name)
		buf.WriteString("[*].")
	}
	buf.WriteString(e.relSubject.String())

	// For consistency we'll also use the splat-like wildcard syntax to
	// represent the final step being either a resource or module call
	// rather than an instance, so we can more easily distinguish the two
	// in the string representation.
	switch e.relSubject.(type) {
	case AbsModuleCall, AbsResource:
		buf.WriteString("[*]")
	}

	return buf.String()
}

// SelectsMoveable returns true if the reciever directly selects the object
// represented by the given address, without any consideration of nesting.
//
// This is a good function to use for deciding whether a specific object
// found in the state should be acted on by a particular move statement.
func (e *MoveEndpointInModule) SelectsMoveable(addr AbsMoveable) bool {
	// Only addresses of the same kind can possibly match. This guarantees
	// that our logic below only needs to deal with combinations of resources
	// and resource instances or with combinations of module calls and
	// module instances.
	if e.ObjectKind() != absMoveableEndpointKind(addr) {
		return false
	}

	// TODO: implement
	return false
}

// CanChainFrom returns true if the reciever describes an address that could
// potentially select an object that the other given address could select.
//
// In other words, this decides whether the move chaining rule applies, if
// the reciever is the "to" from one statement and the other given address
// is the "from" of another statement.
func (e *MoveEndpointInModule) CanChainFrom(other *MoveEndpointInModule) bool {
	// TODO: implement
	return false
}

// NestedWithin returns true if the reciever describes an address that is
// contained within one of the objects that the given other address could
// select.
func (e *MoveEndpointInModule) NestedWithin(other *MoveEndpointInModule) bool {
	// TODO: implement
	return false
}
