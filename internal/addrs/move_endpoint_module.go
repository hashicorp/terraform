package addrs

import (
	"fmt"
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

// SelectsModule returns true if the reciever directly selects either
// the given module or a resource nested directly inside that module.
//
// This is a good function to use to decide which modules in a state
// to consider when processing a particular move statement. For a
// module move the given module itself is what will move, while a
// resource move indicates that we should search each of the resources in
// the given module to see if they match.
func (e *MoveEndpointInModule) SelectsModule(addr ModuleInstance) bool {
	// In order to match the given module path should be at least as
	// long as the path to the module where the move endpoint was defined.
	if len(addr) < len(e.module) {
		return false
	}

	containerPart := addr[:len(e.module)]
	relPart := addr[len(e.module):]

	// The names of all of the steps that align with e.module must match,
	// though the instance keys are wildcards for this part.
	for i := range e.module {
		if containerPart[i].Name != e.module[i] {
			return false
		}
	}

	// The remaining module address steps must match both name and key.
	// The logic for all of these is similar but we will retrieve the
	// module address differently for each type.
	var relMatch ModuleInstance
	switch relAddr := e.relSubject.(type) {
	case ModuleInstance:
		relMatch = relAddr
	case AbsModuleCall:
		// This one requires a little more fuss because the call effectively
		// slices in two the final step of the module address.
		if len(relPart) != len(relAddr.Module)+1 {
			return false
		}
		callPart := relPart[len(relPart)-1]
		if callPart.Name != relAddr.Call.Name {
			return false
		}
	case AbsResource:
		relMatch = relAddr.Module
	case AbsResourceInstance:
		relMatch = relAddr.Module
	default:
		panic(fmt.Sprintf("unhandled relative address type %T", relAddr))
	}

	if len(relPart) != len(relMatch) {
		return false
	}
	for i := range relMatch {
		if relPart[i] != relMatch[i] {
			return false
		}
	}
	return true
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

// MoveDestination considers a an address representing a module
// instance in the context of source and destination move endpoints and then,
// if the module address matches the from endpoint, returns the corresponding
// new module address that the object should move to.
//
// MoveDestination will return false in its second return value if the receiver
// doesn't match fromMatch, indicating that the given move statement doesn't
// apply to this object.
//
// Both of the given endpoints must be from the same move statement and thus
// must have matching object types. If not, MoveDestination will panic.
func (m ModuleInstance) MoveDestination(fromMatch, toMatch *MoveEndpointInModule) (ModuleInstance, bool) {
	return nil, false
}

// MoveDestination considers a an address representing a resource
// in the context of source and destination move endpoints and then,
// if the resource address matches the from endpoint, returns the corresponding
// new resource address that the object should move to.
//
// MoveDestination will return false in its second return value if the receiver
// doesn't match fromMatch, indicating that the given move statement doesn't
// apply to this object.
//
// Both of the given endpoints must be from the same move statement and thus
// must have matching object types. If not, MoveDestination will panic.
func (r AbsResource) MoveDestination(fromMatch, toMatch *MoveEndpointInModule) (AbsResource, bool) {
	return AbsResource{}, false
}

// MoveDestination considers a an address representing a resource
// instance in the context of source and destination move endpoints and then,
// if the instance address matches the from endpoint, returns the corresponding
// new instance address that the object should move to.
//
// MoveDestination will return false in its second return value if the receiver
// doesn't match fromMatch, indicating that the given move statement doesn't
// apply to this object.
//
// Both of the given endpoints must be from the same move statement and thus
// must have matching object types. If not, MoveDestination will panic.
func (r AbsResourceInstance) MoveDestination(fromMatch, toMatch *MoveEndpointInModule) (AbsResourceInstance, bool) {
	return AbsResourceInstance{}, false
}
