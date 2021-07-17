package globalref

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Reference combines an addrs.Reference with the address of the module
// instance or resource instance where it was found.
//
// Because of the design of the Terraform language, our main model of
// references only captures the module-local part of the reference and assumes
// that it's always clear from context which module a reference belongs to.
// That's not true for globalref because our whole purpose is to work across
// module boundaries, and so this package in particular has its own
// representation of references.
type Reference struct {
	// ContainerAddr is always either addrs.ModuleInstance or
	// addrs.AbsResourceInstance. The latter is required if LocalRef's
	// subject is either an addrs.CountAddr or addrs.ForEachAddr, so
	// we can know which resource's repetition expression it's
	// referring to.
	ContainerAddr addrs.Targetable

	// LocalRef is a reference that would be resolved in the context
	// of the module instance or resource instance given in ContainerAddr.
	LocalRef *addrs.Reference
}

func absoluteRef(containerAddr addrs.Targetable, localRef *addrs.Reference) Reference {
	ret := Reference{
		ContainerAddr: containerAddr,
		LocalRef:      localRef,
	}
	// For simplicity's sake, we always reduce the ContainerAddr to be
	// just the module address unless it's a count.index, each.key, or
	// each.value reference, because for anything else it's immaterial
	// which resource it belongs to.
	switch localRef.Subject.(type) {
	case addrs.CountAttr, addrs.ForEachAttr:
		// nothing to do
	default:
		ret.ContainerAddr = ret.ModuleAddr()
	}
	return ret
}

func absoluteRefs(containerAddr addrs.Targetable, refs []*addrs.Reference) []Reference {
	if len(refs) == 0 {
		return nil
	}

	ret := make([]Reference, len(refs))
	for i, ref := range refs {
		ret[i] = absoluteRef(containerAddr, ref)
	}
	return ret
}

// ModuleAddr returns the address of the module where the reference would
// be resolved.
//
// This is either ContainerAddr directly if it's already just a module
// instance, or the module instance part of it if it's a resource instance.
func (r Reference) ModuleAddr() addrs.ModuleInstance {
	switch addr := r.ContainerAddr.(type) {
	case addrs.ModuleInstance:
		return addr
	case addrs.AbsResourceInstance:
		return addr.Module
	default:
		// NOTE: We're intentionally using only a subset of possible
		// addrs.Targetable implementations here, so anything else
		// is invalid.
		panic(fmt.Sprintf("reference has invalid container address type %T", addr))
	}
}

// ResourceAddr returns the address of the resource where the reference
// would be resolved, if there is one.
//
// Because not all references belong to resources, the extra boolean return
// value indicates whether the returned address is valid.
func (r Reference) ResourceAddr() (addrs.AbsResource, bool) {
	switch addr := r.ContainerAddr.(type) {
	case addrs.ModuleInstance:
		return addrs.AbsResource{}, false
	case addrs.AbsResourceInstance:
		return addr.ContainingResource(), true
	default:
		// NOTE: We're intentionally using only a subset of possible
		// addrs.Targetable implementations here, so anything else
		// is invalid.
		panic(fmt.Sprintf("reference has invalid container address type %T", addr))
	}
}

// DebugString returns an internal (but still somewhat Terraform-language-like)
// compact string representation of the reciever, which isn't an address that
// any of our usual address parsers could accept but still captures the
// essence of what the reference represents.
//
// The DebugString result is not suitable for end-user-oriented messages.
//
// DebugString is also not suitable for use as a unique key for a reference,
// because it's ambiguous (between a no-key resource instance and a resource)
// and because it discards the source location information in the LocalRef.
func (r Reference) DebugString() string {
	// As the doc comment insinuates, we don't have any real syntax for
	// "absolute references": references are always local, and targets are
	// always absolute but only include modules and resources.
	return r.ContainerAddr.String() + "::" + r.LocalRef.DisplayString()
}

// addrKey returns the referenceAddrKey value for the item that
// this reference refers to, discarding any source location information.
//
// See the referenceAddrKey doc comment for more information on what this
// is suitable for.
func (r Reference) addrKey() referenceAddrKey {
	// This is a pretty arbitrary bunch of stuff. We include the type here
	// just to differentiate between no-key resource instances and resources.
	return referenceAddrKey(fmt.Sprintf("%s(%T)%s", r.ContainerAddr.String(), r.LocalRef.Subject, r.LocalRef.DisplayString()))
}

// referenceAddrKey is a special string type which conventionally contains
// a unique string representation of the object that a reference refers to,
// although not of the reference itself because it ignores the information
// that would differentiate two different references to the same object.
//
// The actual content of a referenceAddrKey is arbitrary, for internal use
// only. and subject to change in future. We use a named type here only to
// make it easier to see when we're intentionally using strings to uniquely
// identify absolute reference addresses.
type referenceAddrKey string
