// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

type policyTypeKey struct {
	source string
	typ    string
}

// policyObject is an object of a managed resource instance that relationships
// can relate: a row of the candidates table of contract §3.4.
type policyObject struct {
	id   string
	addr string
	typ  policyTypeKey
	op   proto.Operation
	// provider is the address of the resolved provider configuration.
	provider string

	// value is the value that keys read, without marks: the attrs, or the
	// prior attrs of a DELETE.
	value cty.Value
	// origins are the origins of the object's key paths in a plan, by
	// dot-joined key path, other than the values themselves.
	origins map[string][]string
	// reads are what the object's key paths read, by dot-joined key path.
	reads map[string]policyRead
}

// read returns what the key path reads from the object.
func (o *policyObject) read(path []string) policyRead {
	if read, ok := o.reads[strings.Join(path, ".")]; ok {
		return read
	}
	return o.readPath(path)
}

// readPath reads the values at path, with their origins in a plan. Every
// value is its own origin, ADDRESS.KEYPATH, except those of a DELETE, which
// are prior values.
func (o *policyObject) readPath(path []string) policyRead {
	var read policyRead
	if o.value == cty.NilVal {
		read.unknownLen = true
		return read
	}
	read.scalar = policyScalarPath(o.value.Type(), path)
	read.read(o.value, path)
	if o.op == proto.Operation_DELETE || !read.scalar || read.empty() {
		return read
	}
	key := strings.Join(path, ".")
	read.origins = append([]string{o.addr + "." + key}, o.origins[key]...)
	return read
}

// policyRead is what one side of a key reads (contract §3.5): one scalar, or
// a list of scalars in order, with the origins of a scalar.
type policyRead struct {
	// vals are the scalars read, unknown ones included; nulls and "" are
	// left out.
	vals []cty.Value
	// scalar is set if the path reads one primitive value, stepping through
	// no list, set or tuple.
	scalar bool
	// unknownLen is set if a collection or an object on the way isn't
	// known, so there are scalars that aren't in vals.
	unknownLen bool
	origins    []string
}

// read appends the values at path in v. Lists, sets and tuples are stepped
// through as if flat, and null and "" add nothing. Anything that doesn't fit
// the path is read as unknown.
func (r *policyRead) read(v cty.Value, path []string) {
	ty := v.Type()
	if !v.IsKnown() {
		if len(path) == 0 && ty.IsPrimitiveType() {
			r.vals = append(r.vals, v)
		} else {
			r.unknownLen = true
		}
		return
	}
	if v.IsNull() {
		return
	}
	if ty.IsListType() || ty.IsSetType() || ty.IsTupleType() {
		for it := v.ElementIterator(); it.Next(); {
			_, elem := it.Element()
			r.read(elem, path)
		}
		return
	}
	if len(path) == 0 {
		if !ty.IsPrimitiveType() {
			r.unknownLen = true
			return
		}
		if ty == cty.String && v.AsString() == "" {
			return
		}
		r.vals = append(r.vals, v)
		return
	}
	if !ty.IsObjectType() || !ty.HasAttribute(path[0]) {
		r.unknownLen = true
		return
	}
	r.read(v.GetAttr(path[0]), path[1:])
}

// policyScalarPath returns true if a path of a value of type ty reads one
// primitive value, stepping through no list, set or tuple.
func policyScalarPath(ty cty.Type, path []string) bool {
	for _, name := range path {
		if !ty.IsObjectType() || !ty.HasAttribute(name) {
			return false
		}
		ty = ty.AttributeType(name)
	}
	return ty.IsPrimitiveType()
}

// empty returns true if nothing is read: a null or "" scalar, or an empty
// list.
func (r policyRead) empty() bool {
	return len(r.vals) == 0 && !r.unknownLen
}

func (r policyRead) known() bool {
	if r.unknownLen {
		return false
	}
	for _, v := range r.vals {
		if !v.IsKnown() {
			return false
		}
	}
	return true
}

// holds returns true if a key holds between two reads: equal scalars, a
// scalar that equals an element of a list, two lists that are equal element
// by element in order, or, in a plan, a shared origin (contract §3.5).
func (r policyRead) holds(o policyRead) bool {
	if r.empty() || o.empty() {
		return false
	}
	if r.scalar && o.scalar {
		if r.known() && o.known() {
			return policyValueEqual(r.vals[0], o.vals[0])
		}
		for _, a := range r.origins {
			for _, b := range o.origins {
				if a == b {
					return true
				}
			}
		}
		return false
	}
	if r.scalar != o.scalar {
		scalar, list := r, o
		if o.scalar {
			scalar, list = o, r
		}
		if !scalar.known() {
			return false
		}
		for _, v := range list.vals {
			if v.IsKnown() && policyValueEqual(scalar.vals[0], v) {
				return true
			}
		}
		return false
	}
	if !r.known() || !o.known() || len(r.vals) != len(o.vals) {
		return false
	}
	for i := range r.vals {
		if !policyValueEqual(r.vals[i], o.vals[i]) {
			return false
		}
	}
	return true
}

// fails returns true if a key that doesn't hold certainly fails: one side is
// empty, or both sides are wholly known (contract §3.5).
func (r policyRead) fails(o policyRead) bool {
	return r.empty() || o.empty() || (r.known() && o.known())
}

func policyValueEqual(a, b cty.Value) bool {
	return a.Type().Equals(b.Type()) && a.Equals(b).True()
}

// policyRelations relates the instances of a walk and builds the relations
// of its subjects (contract §3.4, §3.5).
type policyRelations struct {
	plan bool

	// trees are the relationship trees by provider source and subject type.
	trees   map[policyTypeKey]*proto.RelationshipTree
	objects map[policyTypeKey][]*policyObject
	byID    map[string]*policyObject
	errored map[policyTypeKey]bool
	// deferred are the deferred instances of each type, with the addresses
	// of their provider configurations, by address.
	deferred map[policyTypeKey]map[string]string
}

func newPolicyRelations(trees []*proto.RelationshipTree, plan bool) *policyRelations {
	r := &policyRelations{
		plan:     plan,
		trees:    make(map[policyTypeKey]*proto.RelationshipTree),
		objects:  make(map[policyTypeKey][]*policyObject),
		byID:     make(map[string]*policyObject),
		errored:  make(map[policyTypeKey]bool),
		deferred: make(map[policyTypeKey]map[string]string),
	}
	for _, tree := range trees {
		r.trees[policyTypeKey{source: tree.ProviderSource, typ: tree.ResourceType}] = tree
	}
	return r
}

func (r *policyRelations) add(obj *policyObject) {
	r.objects[obj.typ] = append(r.objects[obj.typ], obj)
	r.byID[obj.id] = obj
}

// finish sorts the objects and reads their key paths, before subjects are
// evaluated concurrently.
func (r *policyRelations) finish(keyPaths map[policyTypeKey][][]string) {
	for key, objs := range r.objects {
		for _, obj := range objs {
			obj.reads = make(map[string]policyRead, len(keyPaths[key]))
			for _, path := range keyPaths[key] {
				obj.reads[strings.Join(path, ".")] = obj.readPath(path)
			}
		}
	}
}

// subject returns the relations of a subject: one per relationship at the
// top of its type's tree, or nil if its type has no tree. from is the
// subject's object, used if the walk has none with its identifier.
func (r *policyRelations) subject(from *policyObject) []*proto.Relation {
	tree := r.trees[from.typ]
	if tree == nil {
		return nil
	}
	if obj, ok := r.byID[from.id]; ok {
		from = obj
	}
	return r.relations(from, tree.Relationships)
}

// relations returns the relation of each node walked from an object.
func (r *policyRelations) relations(from *policyObject, nodes []*proto.Relationship) []*proto.Relation {
	ret := make([]*proto.Relation, 0, len(nodes))
	for _, node := range nodes {
		rel := &proto.Relation{Relationship: node.Name, Resources: make(map[string]*proto.RelatedResource)}
		ret = append(ret, rel)

		var fromIsKey bool
		var otherType string
		switch from.typ.typ {
		case node.KeyType:
			fromIsKey, otherType = true, node.ValueType
		case node.ValueType:
			otherType = node.KeyType
		default:
			rel.Completeness = proto.Completeness_ERROR_COMPLETENESS
			continue
		}

		key := policyTypeKey{source: node.ProviderSource, typ: otherType}
		undecided := false
		for _, obj := range r.objects[key] {
			if obj.addr == from.addr || obj.provider != from.provider {
				continue
			}
			switch policyMatches(node, fromIsKey, from, obj) {
			case policyRelated:
				rel.Resources[obj.id] = &proto.RelatedResource{
					Operation:     obj.op,
					Relationships: r.relations(obj, node.Relationships),
				}
			case policyUndecided:
				undecided = true
			}
		}
		rel.Completeness, rel.Deferred = r.completeness(key, from, undecided)
	}
	return ret
}

// policyMatch is how a candidate relates to the object a relation is
// walked from.
type policyMatch int

const (
	policyUnrelated policyMatch = iota
	// policyUndecided: not related, but could be after apply.
	policyUndecided
	policyRelated
)

// policyMatches returns policyRelated if every key of some match of a
// definition holds between from and m, and policyUndecided if no key of some
// match certainly fails.
func policyMatches(def *proto.Relationship, fromIsKey bool, from, m *policyObject) policyMatch {
	ret := policyUnrelated
	for _, match := range def.Matches {
		result := policyRelated
		for _, key := range match.Keys {
			fromPath, mPath := key.KeyPath, key.ValuePath
			if !fromIsKey {
				fromPath, mPath = mPath, fromPath
			}
			fromRead, mRead := from.read(policyKeyPathNames(fromPath)), m.read(policyKeyPathNames(mPath))
			if fromRead.holds(mRead) {
				continue
			}
			if fromRead.fails(mRead) {
				result = policyUnrelated
				break
			}
			result = policyUndecided
		}
		if len(match.Keys) == 0 {
			result = policyUnrelated
		}
		ret = max(ret, result)
		if ret == policyRelated {
			break
		}
	}
	return ret
}

// completeness returns the completeness of a relation to the given type
// walked from an object, and the deferred addresses with DEFERRED. undecided
// is set if a candidate is undecided.
func (r *policyRelations) completeness(key policyTypeKey, from *policyObject, undecided bool) (proto.Completeness, []string) {
	if r.errored[key] {
		return proto.Completeness_ERROR_COMPLETENESS, nil
	}
	var deferred []string
	for addr, provider := range r.deferred[key] {
		if addr != from.addr && provider == from.provider {
			deferred = append(deferred, addr)
		}
	}
	if len(deferred) > 0 {
		sort.Strings(deferred)
		return proto.Completeness_DEFERRED_COMPLETENESS, deferred
	}
	if r.plan && undecided {
		return proto.Completeness_PLAN_COMPLETENESS, nil
	}
	return proto.Completeness_COMPLETE_COMPLETENESS, nil
}
