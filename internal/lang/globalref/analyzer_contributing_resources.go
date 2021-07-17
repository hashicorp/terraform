package globalref

import (
	"sort"

	"github.com/hashicorp/terraform/internal/addrs"
)

// ContributingResources analyzes all of the given references and
// for each one tries to walk backwards through any named values to find all
// resources whose values contributed either directly or indirectly to any of
// them.
//
// This is a wrapper around ContributingResourceReferences which simplifies
// the result to only include distinct resource addresses, not full references.
// If the configuration includes several different references to different
// parts of a resource, ContributingResources will not preserve that detail.
func (a *Analyzer) ContributingResources(refs ...Reference) []addrs.AbsResource {
	retRefs := a.ContributingResourceReferences(refs...)
	if len(retRefs) == 0 {
		return nil
	}

	uniq := make(map[string]addrs.AbsResource, len(refs))
	for _, ref := range retRefs {
		if addr, ok := resourceForAddr(ref.LocalRef.Subject); ok {
			moduleAddr := ref.ModuleAddr()
			absAddr := addr.Absolute(moduleAddr)
			uniq[absAddr.String()] = absAddr
		}
	}
	ret := make([]addrs.AbsResource, 0, len(uniq))
	for _, addr := range uniq {
		ret = append(ret, addr)
	}
	sort.Slice(ret, func(i, j int) bool {
		// We only have a sorting function for resource _instances_, but
		// it'll do well enough if we just pretend we have no-key instances.
		return ret[i].Instance(addrs.NoKey).Less(ret[j].Instance(addrs.NoKey))
	})
	return ret
}

// ContributingResourceReferences analyzes all of the given references and
// for each one tries to walk backwards through any named values to find all
// references to resource attributes that contributed either directly or
// indirectly to any of them.
//
// This is a global operation that can be potentially quite expensive for
// complex configurations.
func (a *Analyzer) ContributingResourceReferences(refs ...Reference) []Reference {
	// Our methodology here is to keep digging through MetaReferences
	// until we've visited everything we encounter directly or indirectly,
	// and keep track of any resources we find along the way.

	// We'll aggregate our result here, using the string representations of
	// the resources as keys to avoid returning the same one more than once.
	found := make(map[referenceAddrKey]Reference)

	// We might encounter the same object multiple times as we walk,
	// but we won't learn anything more by traversing them again and so we'll
	// just skip them instead.
	visitedObjects := make(map[referenceAddrKey]struct{})

	// A queue of objects we still need to visit.
	// Note that if we find multiple references to the same object then we'll
	// just arbitrary choose any one of them, because for our purposes here
	// it's immaterial which reference we actually followed.
	pendingObjects := make(map[referenceAddrKey]Reference)

	// Initial state: identify any directly-mentioned resources and
	// queue up any named values we refer to.
	for _, ref := range refs {
		if _, ok := resourceForAddr(ref.LocalRef.Subject); ok {
			found[ref.addrKey()] = ref
		}
		pendingObjects[ref.addrKey()] = ref
	}

	for len(pendingObjects) > 0 {
		// Note: we modify this map while we're iterating over it, which means
		// that anything we add might be either visited within a later
		// iteration of the inner loop or in a later iteration of the outer
		// loop, but we get the correct result either way because we keep
		// working until we've fully depleted the queue.
		for key, ref := range pendingObjects {
			delete(pendingObjects, key)

			// We do this _before_ the visit below just in case this is an
			// invalid config with a self-referential local value, in which
			// case we'll just silently ignore the self reference for our
			// purposes here, and thus still eventually converge (albeit
			// with an incomplete answer).
			visitedObjects[key] = struct{}{}

			moreRefs := a.MetaReferences(ref)
			for _, newRef := range moreRefs {
				if _, ok := resourceForAddr(newRef.LocalRef.Subject); ok {
					found[newRef.addrKey()] = newRef
				}

				newKey := newRef.addrKey()
				if _, visited := visitedObjects[newKey]; !visited {
					pendingObjects[newKey] = newRef
				}
			}
		}
	}

	if len(found) == 0 {
		return nil
	}

	ret := make([]Reference, 0, len(found))
	for _, ref := range found {
		ret = append(ret, ref)
	}
	return ret
}

func resourceForAddr(addr addrs.Referenceable) (addrs.Resource, bool) {
	switch addr := addr.(type) {
	case addrs.Resource:
		return addr, true
	case addrs.ResourceInstance:
		return addr.Resource, true
	default:
		return addrs.Resource{}, false
	}
}
