// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sort"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

type policyTypeKey struct {
	source string
	typ    string
}

// PolicyProvider is the provider configuration of a resource instance, for
// the provider rule (contract §3.5): two instances can only be related if
// they use the same provider configuration. Instances with the same
// configuration share one PolicyProvider.
type PolicyProvider struct {
	// Addr identifies the configuration: its address, or in Terraform
	// Stacks the stack address of a provider configuration instance.
	Addr string
	// Config is the configuration's value without marks if it's wholly
	// known, and cty.NilVal otherwise.
	Config cty.Value
}

// same returns true if p and o are the same configuration: the same address,
// or known and equal values.
func (p *PolicyProvider) same(o *PolicyProvider) bool {
	if p.Addr == o.Addr {
		return true
	}
	return p.Config != cty.NilVal && o.Config != cty.NilVal && p.Config.RawEquals(o.Config)
}

// differs returns true if p and o are known to be different configurations.
func (p *PolicyProvider) differs(o *PolicyProvider) bool {
	return !p.same(o) && p.Config != cty.NilVal && o.Config != cty.NilVal
}

// policyObject is an object of a managed resource instance that relationships
// can relate: a row of the candidates table of contract §3.4.
type policyObject struct {
	addr    string
	deposed string
	// old is set for the object that a replace deletes.
	old          bool
	typ          policyTypeKey
	op           proto.Operation
	modulePath   string
	providerType string
	provider     *PolicyProvider

	attrs, priorAttrs *proto.ResourceAttributes

	// value is the value that keys read, without marks: the attrs, or the
	// prior attrs of a DELETE.
	value cty.Value
	// origins are the origins of the object's key paths in a plan, by
	// dot-joined key path. vars are the leaves of key paths whose values are
	// root module variables, when the walk follows them.
	origins map[string][]PolicyOrigin
	vars    map[string][]PolicyVariableOrigin
	// bags are what the object's key paths read, by dot-joined key path.
	bags map[string]policyBag
}

func (o *policyObject) member() *proto.RelatedMember {
	return &proto.RelatedMember{
		Address:        o.addr,
		DeposedKey:     o.deposed,
		OldObject:      o.old,
		Type:           o.typ.typ,
		ProviderType:   o.providerType,
		ProviderSource: o.typ.source,
		ModulePath:     o.modulePath,
		Operation:      o.op,
		Attrs:          o.attrs,
		PriorAttrs:     o.priorAttrs,
	}
}

// bag returns what the key path reads from the object.
func (o *policyObject) bag(path []string) policyBag {
	if bag, ok := o.bags[strings.Join(path, ".")]; ok {
		return bag
	}
	return o.readBag(path)
}

// readBag reads the values at path, with their origins. The values of a
// DELETE are prior values, which have no origins.
func (o *policyObject) readBag(path []string) policyBag {
	bag := policyBag{known: make(map[string]bool)}
	if o.value == cty.NilVal {
		bag.unknown = true
		return bag
	}
	bag.read(o.value, path)
	if o.op == proto.Operation_DELETE || (len(bag.known) == 0 && !bag.unknown) {
		return bag
	}
	key := strings.Join(path, ".")
	bag.self = o.addr + "." + key
	bag.origins = make(map[string]bool)
	bag.unindexed = make(map[string]bool)
	for _, origin := range o.origins[key] {
		bag.origins[origin.id()] = true
		bag.unindexed[origin.unindexed()] = true
	}
	return bag
}

// policyBag is what one side of a key reads: a set of primitive values, and
// their origins.
type policyBag struct {
	known   map[string]bool
	unknown bool
	// self identifies the values themselves, ADDRESS.KEYPATH: every value
	// is its own origin.
	self      string
	origins   map[string]bool
	unindexed map[string]bool
}

// read adds the values at path in v. Lists, sets and tuples flatten at any
// depth, and null and "" add nothing. Anything that doesn't fit the path is
// read as unknown.
func (b *policyBag) read(v cty.Value, path []string) {
	if !v.IsKnown() {
		b.unknown = true
		return
	}
	if v.IsNull() {
		return
	}
	ty := v.Type()
	if ty.IsListType() || ty.IsSetType() || ty.IsTupleType() {
		for it := v.ElementIterator(); it.Next(); {
			_, elem := it.Element()
			b.read(elem, path)
		}
		return
	}
	if len(path) == 0 {
		switch ty {
		case cty.String:
			if s := v.AsString(); s != "" {
				b.known["string:"+s] = true
			}
		case cty.Number:
			b.known["number:"+v.AsBigFloat().Text('g', -1)] = true
		case cty.Bool:
			if v.True() {
				b.known["bool:true"] = true
			} else {
				b.known["bool:false"] = true
			}
		default:
			b.unknown = true
		}
		return
	}
	if !ty.IsObjectType() || !ty.HasAttribute(path[0]) {
		b.unknown = true
		return
	}
	b.read(v.GetAttr(path[0]), path[1:])
}

// holds returns true if a key holds between two bags: they share a known
// value or an origin.
func (b policyBag) holds(o policyBag) bool {
	for id := range b.origins {
		if o.origins[id] {
			return true
		}
	}
	if (o.self != "" && b.unindexed[o.self]) || (b.self != "" && o.unindexed[b.self]) {
		return true
	}
	for value := range b.known {
		if o.known[value] {
			return true
		}
	}
	return false
}

// PolicyInstances are the instances of a walk that relationships can relate.
type PolicyInstances struct {
	objects []*policyObject
	// errored are the types whose instances aren't all known.
	errored map[policyTypeKey]bool
	// deferred are the deferred instances of each type, with their provider
	// configurations, by address.
	deferred map[policyTypeKey]map[string]*PolicyProvider

	// Providers are the provider configurations of the instances, by their
	// address in the walk.
	Providers map[string]*PolicyProvider
}

// prefix qualifies the addresses of the instances, their origins and the
// addresses of their provider configurations.
func (p *PolicyInstances) prefix(prefix string) {
	for _, obj := range p.objects {
		obj.addr = prefix + obj.addr
		for _, origins := range obj.origins {
			for i := range origins {
				origins[i].Address = prefix + origins[i].Address
			}
		}
	}
	for key, deferred := range p.deferred {
		prefixed := make(map[string]*PolicyProvider, len(deferred))
		for addr, provider := range deferred {
			prefixed[prefix+addr] = provider
		}
		p.deferred[key] = prefixed
	}
	for _, provider := range p.Providers {
		provider.Addr = prefix + provider.Addr
	}
}

// ResolveVariables adds origins to the leaves of key paths whose values are
// root module variables: resolve returns the origins of a leaf.
func (p *PolicyInstances) ResolveVariables(resolve func(PolicyVariableOrigin) []PolicyOrigin) {
	for _, obj := range p.objects {
		for path, leaves := range obj.vars {
			for _, leaf := range leaves {
				if obj.origins == nil {
					obj.origins = make(map[string][]PolicyOrigin)
				}
				obj.origins[path] = append(obj.origins[path], resolve(leaf)...)
			}
		}
	}
}

// PolicyRelations relates instances and builds the related members of
// subjects (contract §3.4, §3.5).
type PolicyRelations struct {
	plan bool
	// errored makes every hop ERROR.
	errored bool

	// defs are the definitions by provider source and name.
	defs     map[policyTypeKey]*proto.RelationshipDefinition
	usages   map[policyTypeKey]*proto.RelationshipUsage
	objects  map[policyTypeKey][]*policyObject
	byAddr   map[string][]*policyObject
	erroredT map[policyTypeKey]bool
	deferred map[policyTypeKey]map[string]*PolicyProvider
}

// NewPolicyRelations returns the relations of the given instances for the
// relationships. plan is set in a plan, and errored when the instances
// aren't all known, e.g. because the operation failed.
func NewPolicyRelations(rels policy.Relationships, plan, errored bool, insts ...*PolicyInstances) *PolicyRelations {
	r := &PolicyRelations{
		plan:     plan,
		errored:  errored,
		defs:     make(map[policyTypeKey]*proto.RelationshipDefinition),
		usages:   make(map[policyTypeKey]*proto.RelationshipUsage),
		objects:  make(map[policyTypeKey][]*policyObject),
		byAddr:   make(map[string][]*policyObject),
		erroredT: make(map[policyTypeKey]bool),
		deferred: make(map[policyTypeKey]map[string]*PolicyProvider),
	}
	for _, def := range rels.Definitions {
		r.defs[policyTypeKey{source: def.ProviderSource, typ: def.Name}] = def
	}
	for _, usage := range rels.Usages {
		r.usages[policyTypeKey{source: usage.ProviderSource, typ: usage.Type}] = usage
	}
	for _, inst := range insts {
		for _, obj := range inst.objects {
			r.objects[obj.typ] = append(r.objects[obj.typ], obj)
			r.byAddr[obj.addr] = append(r.byAddr[obj.addr], obj)
		}
		for key := range inst.errored {
			r.erroredT[key] = true
		}
		for key, deferred := range inst.deferred {
			if r.deferred[key] == nil {
				r.deferred[key] = make(map[string]*PolicyProvider)
			}
			for addr, provider := range deferred {
				r.deferred[key][addr] = provider
			}
		}
	}
	for _, objs := range r.objects {
		sort.Slice(objs, func(i, j int) bool {
			a, b := objs[i], objs[j]
			if a.addr != b.addr {
				return a.addr < b.addr
			}
			if a.deposed != b.deposed {
				return a.deposed < b.deposed
			}
			return !a.old && b.old
		})
	}

	// The bags are read once, before subjects are evaluated concurrently.
	for key, paths := range policyKeyPathsByType(rels) {
		for _, obj := range r.objects[key] {
			obj.bags = make(map[string]policyBag, len(paths))
			for _, path := range paths {
				obj.bags[strings.Join(path, ".")] = obj.readBag(path)
			}
		}
	}
	return r
}

// Related returns the related members of a subject: one entry per hop of
// the usage of its type, or nil if its type has no usage. before and after
// are the subject's values, used if the walk has no object for it.
func (r *PolicyRelations) Related(meta *proto.PolicyEvaluateResourceRequest_ResourceMetadata, typ string, before, after cty.Value) []*proto.RelatedMembers {
	key := policyTypeKey{source: meta.ProviderSource, typ: typ}
	usage := r.usages[key]
	if usage == nil {
		return nil
	}
	isDelete := meta.Operation == proto.Operation_DELETE
	var from *policyObject
	for _, obj := range r.byAddr[meta.Address] {
		if obj.deposed == meta.DeposedKey && (obj.op == proto.Operation_DELETE) == isDelete {
			from = obj
			break
		}
	}
	if from == nil {
		// The object failed to encode, so its provider configuration isn't
		// known either.
		from = &policyObject{addr: meta.Address, deposed: meta.DeposedKey, typ: key, op: meta.Operation, provider: &PolicyProvider{}}
		from.value = after
		if isDelete {
			from.value = before
		}
		if from.value != cty.NilVal {
			from.value, _ = from.value.UnmarkDeep()
		}
	}
	return r.related(from, usage.Hops)
}

// related returns the members of each hop walked from an instance.
func (r *PolicyRelations) related(from *policyObject, hops []*proto.RelationshipHop) []*proto.RelatedMembers {
	ret := make([]*proto.RelatedMembers, 0, len(hops))
	for _, hop := range hops {
		members := &proto.RelatedMembers{Relationship: hop.Relationship}
		ret = append(ret, members)

		def := r.defs[policyTypeKey{source: from.typ.source, typ: hop.Relationship}]
		var fromIsKey bool
		var memberType string
		switch {
		case def == nil:
			members.Completeness = proto.Completeness_ERROR_COMPLETENESS
			continue
		case def.KeyResource == from.typ.typ:
			fromIsKey, memberType = true, def.ValueResource
		case def.ValueResource == from.typ.typ:
			memberType = def.KeyResource
		default:
			members.Completeness = proto.Completeness_ERROR_COMPLETENESS
			continue
		}

		key := policyTypeKey{source: from.typ.source, typ: memberType}
		for _, obj := range r.objects[key] {
			if obj.addr == from.addr || !from.provider.same(obj.provider) || !policyMatches(def, fromIsKey, from, obj) {
				continue
			}
			member := obj.member()
			member.Related = r.related(obj, hop.Hops)
			members.Members = append(members.Members, member)
		}
		members.Completeness, members.Deferred = r.completeness(key, from)
	}
	return ret
}

// policyMatches returns true if every key of some alternative of a
// definition holds between from and m.
func policyMatches(def *proto.RelationshipDefinition, fromIsKey bool, from, m *policyObject) bool {
	for _, alt := range def.Match {
		holds := true
		for _, key := range alt.Keys {
			fromPath, mPath := key.KeyPath, key.ValuePath
			if !fromIsKey {
				fromPath, mPath = mPath, fromPath
			}
			if !from.bag(policyKeyPathNames(fromPath)).holds(m.bag(policyKeyPathNames(mPath))) {
				holds = false
				break
			}
		}
		if holds {
			return true
		}
	}
	return false
}

// completeness returns the completeness of a hop to the given type walked
// from an instance, and the deferred addresses with DEFERRED.
func (r *PolicyRelations) completeness(key policyTypeKey, from *policyObject) (proto.Completeness, []string) {
	if r.errored || r.erroredT[key] {
		return proto.Completeness_ERROR_COMPLETENESS, nil
	}
	var deferred []string
	for addr, provider := range r.deferred[key] {
		if addr != from.addr && !from.provider.differs(provider) {
			deferred = append(deferred, addr)
		}
	}
	if len(deferred) > 0 {
		sort.Strings(deferred)
		return proto.Completeness_DEFERRED_COMPLETENESS, deferred
	}
	if r.plan {
		return proto.Completeness_PLAN_COMPLETENESS, nil
	}
	return proto.Completeness_COMPLETE_COMPLETENESS, nil
}
