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

// matchModuleInstancePrefix is an internal helper to decide whether the given
// module instance address refers to either the module where the move endpoint
// was declared or some descendent of that module.
//
// If so, it will split the given address into two parts: the "prefix" part
// which corresponds with the module where the statement was declared, and
// the "relative" part which is the remainder that the relSubject of the
// statement might match against.
//
// The second return value is another example of our light abuse of
// ModuleInstance to represent _relative_ module references rather than
// absolute: it's a module instance address relative to the same return value.
// Because the exported idea of ModuleInstance represents only _absolute_
// module instance addresses, we mustn't expose that value through any exported
// API.
func (e *MoveEndpointInModule) matchModuleInstancePrefix(instAddr ModuleInstance) (ModuleInstance, ModuleInstance, bool) {
	if len(e.module) > len(instAddr) {
		return nil, nil, false // to short to possibly match
	}
	for i := range e.module {
		if e.module[i] != instAddr[i].Name {
			return nil, nil, false
		}
	}
	// If we get here then we have a match, so we'll slice up the input
	// to produce the prefix and match segments.
	return instAddr[:len(e.module)], instAddr[len(e.module):], true
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
	// NOTE: This implementation assumes the invariant that fromMatch and
	// toMatch both belong to the same configuration statement, and thus they
	// will both have the same address type and the same declaration module.

	// The root module instance is not itself moveable.
	if m.IsRoot() {
		return nil, false
	}

	// The two endpoints must either be module call or module instance
	// addresses, or else this statement can never match.
	if fromMatch.ObjectKind() != MoveEndpointModule {
		return nil, false
	}

	// The rest of our work will be against the part of the reciever that's
	// relative to the declaration module. mRel is a weird abuse of
	// ModuleInstance that represents a relative module address, similar to
	// what we do for MoveEndpointInModule.relSubject.
	mPrefix, mRel, match := fromMatch.matchModuleInstancePrefix(m)
	if !match {
		return nil, false
	}

	// Our next goal is to split mRel into two parts: the match (if any) and
	// the suffix. Our result will then replace the match with the replacement
	// in toMatch while preserving the prefix and suffix.
	var mSuffix, mNewMatch ModuleInstance

	switch relSubject := fromMatch.relSubject.(type) {
	case ModuleInstance:
		if len(relSubject) > len(mRel) {
			return nil, false // too short to possibly match
		}
		for i := range relSubject {
			if relSubject[i] != mRel[i] {
				return nil, false // this step doesn't match
			}
		}
		// If we get to here then we've found a match. Since the statement
		// addresses are already themselves ModuleInstance fragments we can
		// just slice out the relevant parts.
		mNewMatch = toMatch.relSubject.(ModuleInstance)
		mSuffix = mRel[len(relSubject):]
	case AbsModuleCall:
		// The module instance part of relSubject must be a prefix of
		// mRel, and mRel must be at least one step longer to account for
		// the call step itself.
		if len(relSubject.Module) > len(mRel)-1 {
			return nil, false
		}
		for i := range relSubject.Module {
			if relSubject.Module[i] != mRel[i] {
				return nil, false // this step doesn't match
			}
		}
		// The call name must also match the next step of mRel, after
		// the relSubject.Module prefix.
		callStep := mRel[len(relSubject.Module)]
		if callStep.Name != relSubject.Call.Name {
			return nil, false
		}
		// If we get to here then we've found a match. We need to construct
		// a new mNewMatch that's an instance of the "new" relSubject with
		// the same key as our call.
		mNewMatch = toMatch.relSubject.(AbsModuleCall).Instance(callStep.InstanceKey)
		mSuffix = mRel[len(relSubject.Module)+1:]
	default:
		panic("invalid address type for module-kind move endpoint")
	}

	ret := make(ModuleInstance, 0, len(mPrefix)+len(mNewMatch)+len(mSuffix))
	ret = append(ret, mPrefix...)
	ret = append(ret, mNewMatch...)
	ret = append(ret, mSuffix...)
	return ret, true
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
	switch fromMatch.ObjectKind() {
	case MoveEndpointModule:
		// If we've moving a module then any resource inside that module
		// moves too.
		fromMod := r.Module
		toMod, match := fromMod.MoveDestination(fromMatch, toMatch)
		if !match {
			return AbsResource{}, false
		}
		return r.Resource.Absolute(toMod), true

	case MoveEndpointResource:
		fromRelSubject, ok := fromMatch.relSubject.(AbsResource)
		if !ok {
			// The only other possible type for a resource move is
			// AbsResourceInstance, and that can never match an AbsResource.
			return AbsResource{}, false
		}

		// fromMatch can only possibly match the reciever if the resource
		// portions are identical, regardless of the module paths.
		if fromRelSubject.Resource != r.Resource {
			return AbsResource{}, false
		}

		// The module path portion of relSubject must have a prefix that
		// matches the module where our endpoints were declared.
		mPrefix, mRel, match := fromMatch.matchModuleInstancePrefix(r.Module)
		if !match {
			return AbsResource{}, false
		}

		// The remaining steps of the module path must _exactly_ match
		// the relative module path in the "fromMatch" address.
		if len(mRel) != len(fromRelSubject.Module) {
			return AbsResource{}, false // can't match if lengths are different
		}
		for i := range mRel {
			if mRel[i] != fromRelSubject.Module[i] {
				return AbsResource{}, false // all of the steps must match
			}
		}

		// If we got here then we have a match, and so our result is the
		// module instance where the statement was declared (mPrefix) followed
		// by the "to" relative address in toMatch.
		toRelSubject := toMatch.relSubject.(AbsResource)
		var mNew ModuleInstance
		if len(mPrefix) > 0 || len(toRelSubject.Module) > 0 {
			mNew = make(ModuleInstance, 0, len(mPrefix)+len(toRelSubject.Module))
			mNew = append(mNew, mPrefix...)
			mNew = append(mNew, toRelSubject.Module...)
		}
		ret := toRelSubject.Resource.Absolute(mNew)
		return ret, true

	default:
		panic("unexpected object kind")
	}
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
	switch fromMatch.ObjectKind() {
	case MoveEndpointModule:
		// If we've moving a module then any resource inside that module
		// moves too.
		fromMod := r.Module
		toMod, match := fromMod.MoveDestination(fromMatch, toMatch)
		if !match {
			return AbsResourceInstance{}, false
		}
		return r.Resource.Absolute(toMod), true

	case MoveEndpointResource:
		switch fromMatch.relSubject.(type) {
		case AbsResource:
			oldResource := r.ContainingResource()
			newResource, match := oldResource.MoveDestination(fromMatch, toMatch)
			if !match {
				return AbsResourceInstance{}, false
			}
			return newResource.Instance(r.Resource.Key), true
		case AbsResourceInstance:
			fromRelSubject, ok := fromMatch.relSubject.(AbsResourceInstance)
			if !ok {
				// The only other possible type for a resource move is
				// AbsResourceInstance, and that can never match an AbsResource.
				return AbsResourceInstance{}, false
			}

			// fromMatch can only possibly match the reciever if the resource
			// portions are identical, regardless of the module paths.
			if fromRelSubject.Resource != r.Resource {
				return AbsResourceInstance{}, false
			}

			// The module path portion of relSubject must have a prefix that
			// matches the module where our endpoints were declared.
			mPrefix, mRel, match := fromMatch.matchModuleInstancePrefix(r.Module)
			if !match {
				return AbsResourceInstance{}, false
			}

			// The remaining steps of the module path must _exactly_ match
			// the relative module path in the "fromMatch" address.
			if len(mRel) != len(fromRelSubject.Module) {
				return AbsResourceInstance{}, false // can't match if lengths are different
			}
			for i := range mRel {
				if mRel[i] != fromRelSubject.Module[i] {
					return AbsResourceInstance{}, false // all of the steps must match
				}
			}

			// If we got here then we have a match, and so our result is the
			// module instance where the statement was declared (mPrefix) followed
			// by the "to" relative address in toMatch.
			toRelSubject := toMatch.relSubject.(AbsResourceInstance)
			var mNew ModuleInstance
			if len(mPrefix) > 0 || len(toRelSubject.Module) > 0 {
				mNew = make(ModuleInstance, 0, len(mPrefix)+len(toRelSubject.Module))
				mNew = append(mNew, mPrefix...)
				mNew = append(mNew, toRelSubject.Module...)
			}
			ret := toRelSubject.Resource.Absolute(mNew)
			return ret, true
		default:
			panic("invalid address type for resource-kind move endpoint")
		}
	default:
		panic("unexpected object kind")
	}
}
