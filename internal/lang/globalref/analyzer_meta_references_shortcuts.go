package globalref

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang"
)

// ReferencesFromOutputValue returns all of the direct references from the
// value expression of the given output value. It doesn't include any indirect
// references.
func (a *Analyzer) ReferencesFromOutputValue(addr addrs.AbsOutputValue) []Reference {
	mc := a.ModuleConfig(addr.Module)
	if mc == nil {
		return nil
	}
	oc := mc.Outputs[addr.OutputValue.Name]
	if oc == nil {
		return nil
	}
	refs, _ := lang.ReferencesInExpr(oc.Expr)
	return absoluteRefs(addr.Module, refs)
}

// ReferencesFromResource returns all of the direct references from the
// definition of the resource instance at the given address. It doesn't
// include any indirect references.
//
// The result doesn't directly include references from a "count" or "for_each"
// expression belonging to the associated resource, but it will include any
// references to count.index, each.key, or each.value that appear in the
// expressions which you can then, if you wish, resolve indirectly using
// Analyzer.MetaReferences. Alternatively, you can use
// Analyzer.ReferencesFromResourceRepetition to get that same result directly.
func (a *Analyzer) ReferencesFromResourceInstance(addr addrs.AbsResourceInstance) []Reference {
	// Using MetaReferences for this is kinda overkill, since
	// lang.ReferencesInBlock would be sufficient really, but
	// this ensures we keep consistent in how we build the
	// resulting absolute references and otherwise aside from
	// some extra overhead this call boils down to a call to
	// lang.ReferencesInBlock anyway.
	fakeRef := Reference{
		ContainerAddr: addr.Module,
		LocalRef: &addrs.Reference{
			Subject: addr.Resource,
		},
	}
	return a.MetaReferences(fakeRef)
}

// ReferencesFromResourceRepetition returns the references from the given
// resource's for_each or count expression, or an empty set if the resource
// doesn't use repetition.
//
// This is a special-case sort of helper for use in situations where an
// expression might refer to count.index, each.key, or each.value, and thus
// we say that it depends indirectly on the repetition expression.
func (a *Analyzer) ReferencesFromResourceRepetition(addr addrs.AbsResource) []Reference {
	modCfg := a.ModuleConfig(addr.Module)
	if modCfg == nil {
		return nil
	}
	rc := modCfg.ResourceByAddr(addr.Resource)
	if rc == nil {
		return nil
	}

	// We're assuming here that resources can either have count or for_each,
	// but never both, because that's a requirement enforced by the language
	// decoder. But we'll assert it just to make sure we catch it if that
	// changes for some reason.
	if rc.ForEach != nil && rc.Count != nil {
		panic(fmt.Sprintf("%s has both for_each and count", addr))
	}

	switch {
	case rc.ForEach != nil:
		refs, _ := lang.ReferencesInExpr(rc.ForEach)
		return absoluteRefs(addr.Module, refs)
	case rc.Count != nil:
		refs, _ := lang.ReferencesInExpr(rc.Count)
		return absoluteRefs(addr.Module, refs)
	default:
		return nil
	}
}
