// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/lang/ctyutil"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/policy"
)

// followRelationships returns a Relation for each of the given relationships, with the
// resources related to the given resource, and follows the nested
// relationships from each of them. A resource that may or may not be related
// makes the Relation incomplete.
func (r *policyWalkState) followRelationships(resource *policyResource, relationships []policy.Relationship) []policy.Relation {
	var ret []policy.Relation
	for _, rel := range relationships {
		// work out what side of the relationship the other type is
		otherType := rel.KeyType
		if resource.Addr.Resource.Resource.Type == rel.KeyType {
			otherType = rel.ValueType
		}

		relation := policy.Relation{
			Relationship: rel.Name,
			Resources:    make(map[string][]policy.Relation),
			Complete:     !r.hasUnknowns(otherType),
		}

		for _, other := range r.typeResourceMap[otherType] {
			// Exclude self before comparing unknown values, so it cannot make a
			// Relation incomplete.
			if other.Addr.Equal(resource.Addr) {
				continue
			}
			related := r.resourcesRelated(resource, other, rel)
			switch {
			case !related.IsKnown():
				relation.Complete = false
			case related.True():
				relation.Resources[other.Addr.String()] = r.followRelationships(other, rel.Relationships)
			}
		}
		ret = append(ret, relation)
	}
	return ret
}

// resourcesRelated returns True when the two resources are related by the given
// relationship, False when they aren't, and an unknown bool when Terraform
// can't tell. Two resources are related when every key of any one match is
// equal. Same-type relationships check both orientations of each full match.
func (r *policyWalkState) resourcesRelated(a, b *policyResource, rel policy.Relationship) cty.Value {
	ret := cty.False
	var keyResource, valueResource *policyResource
	if a.Addr.Resource.Resource.Type == rel.KeyType {
		keyResource, valueResource = a, b
	} else {
		keyResource, valueResource = b, a
	}
	for _, match := range rel.Matches {
		ret = ret.Or(r.matchResources(keyResource, valueResource, match))

		// handle same-type relationships
		if rel.KeyType == rel.ValueType {
			ret = ret.Or(r.matchResources(valueResource, keyResource, match))
		}
	}
	return ret
}

// matchResources compares all keys in one match, in the given orientation.
func (r *policyWalkState) matchResources(keyResource, valueResource *policyResource, match policy.Match) cty.Value {
	matched := cty.True
	for _, key := range match {
		matched = matched.And(r.checkEquality(keyResource, valueResource, key))
	}
	return matched
}

// checkEquality returns True when the two key values will be equal after the run,
// False when they won't, and an unknown bool when Terraform can't tell.
func (r *policyWalkState) checkEquality(keyResource, valueResource *policyResource, key policy.RelationshipKey) cty.Value {
	a, _ := ctyutil.ApplyPath(keyResource.Value.Raw, key.KeyPath)
	b, _ := ctyutil.ApplyPath(valueResource.Value.Raw, key.ValuePath)

	if a == cty.NilVal || b == cty.NilVal {
		return cty.False
	}
	// Two nulls are never equal, because a resource without the attribute
	// isn't related to every other resource without it.
	if a.IsNull() || b.IsNull() {
		return cty.False
	}

	// When the values decide, the origins don't matter.
	if eq := a.Equals(b); eq.IsKnown() {
		return eq
	}

	// Now compare the attributes by origin.
	aRef := globalref.ResourceAttr{Resource: keyResource.Addr, Attr: key.KeyPath}
	bRef := globalref.ResourceAttr{Resource: valueResource.Addr, Attr: key.ValuePath}
	if (a.Range().DefinitelyNotNull() || b.Range().DefinitelyNotNull()) && r.shareOrigin(aRef, bRef) {
		return cty.True
	}
	return cty.UnknownVal(cty.Bool)
}
