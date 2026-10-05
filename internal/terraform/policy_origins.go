// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

// originLookup gives originsFor the information about referenced resource
// instances it needs to decide whether an origin is sound.
type originLookup interface {
	// attrType returns the type of the attribute of a managed resource type
	// at path. ok is false unless path names a primitive, non-write-only
	// attribute through attribute steps only.
	attrType(provider addrs.Provider, resType string, path []string) (ty cty.Type, ok bool)

	// plannedValue returns the planned value of a resource instance at path.
	// ok is false if the value is unknown or isn't available.
	plannedValue(addr addrs.AbsResourceInstance, path []string) (val cty.Value, ok bool)

	// attrComputed returns whether the attribute of a managed resource type
	// at path is computed. ok is false if path doesn't name an attribute.
	attrComputed(provider addrs.Provider, resType string, path []string) (computed, ok bool)
}

// originsFor returns the origins of the planned value of a resource instance
// at the given key paths, following contract §5: a static, symbolic
// evaluation of the instance's configuration that only follows expressions
// which preserve the referenced value. When in doubt, it returns no origins
// for a key path, because a wrong origin can produce a wrong policy result.
// For the leaves without an origin it returns why, following P2 contract §4.
// A key path whose planned value is null gets no entry.
//
// overridden reports whether a module instance is overridden by test
// overrides, or is nil without overrides. The outputs of an overridden module
// come from the override instead of its configuration, so nothing in it is
// followed.
func originsFor(cfg *configs.Config, exp *instances.Expander, overridden func(addrs.ModuleInstance) bool, addr addrs.AbsResourceInstance, planned cty.Value, keyPaths [][]string, lookup originLookup) []*proto.KeyOrigins {
	if planned == cty.NilVal || len(keyPaths) == 0 {
		return nil
	}
	planned, _ = planned.UnmarkDeep()
	if planned.IsNull() {
		return nil
	}
	e := &originEval{cfg: cfg, exp: exp, overridden: overridden}
	rec := e.record(addr, planned, lookup)

	var ret []*proto.KeyOrigins
	for _, kp := range keyPaths {
		if len(kp) == 0 {
			continue
		}
		n, countable := policyPlannedLeaves(planned, kp)
		if countable && n == 0 {
			continue
		}
		origins, reasons := []*proto.Origin(nil), []proto.NoOriginReason{proto.NoOriginReason_UNSUPPORTED_NO_ORIGIN_REASON}
		if rec != nil {
			origins, reasons = rec.keyPath(kp, n, countable)
		}
		if len(origins) == 0 && len(reasons) == 0 {
			continue
		}
		ret = append(ret, &proto.KeyOrigins{KeyPath: policyAttrPath(kp), Origins: origins, NoOrigin: reasons})
	}
	return ret
}

// originRecord is a record whose origins are analyzed.
type originRecord struct {
	e        *originEval
	rc       *configs.Resource
	body     *hclsyntax.Body
	provider addrs.Provider
	self     originSym
	planned  cty.Value
	lookup   originLookup
}

// record returns the record of the resource instance at addr, or nil if its
// configuration can't be analyzed: there's no configuration or expansion,
// its module is overridden, it isn't a managed resource, its body isn't in
// native syntax (JSON bodies and bodies merged from override files are
// other types), or its planned value is unknown.
func (e *originEval) record(addr addrs.AbsResourceInstance, planned cty.Value, lookup originLookup) *originRecord {
	if e.cfg == nil || e.exp == nil || e.moduleOverridden(addr.Module) || !planned.IsKnown() {
		return nil
	}
	modCfg := e.cfg.DescendantForInstance(addr.Module)
	if modCfg == nil {
		return nil
	}
	rc := modCfg.Module.ResourceByAddr(addr.Resource.Resource)
	if rc == nil || rc.Mode != addrs.ManagedResourceMode {
		return nil
	}
	body, ok := rc.Config.(*hclsyntax.Body)
	if !ok {
		return nil
	}
	scope := &originScope{mod: addr.Module, cfg: modCfg, rep: e.resourceRepetition(rc, addr)}
	return &originRecord{
		e:        e,
		rc:       rc,
		body:     body,
		provider: modCfg.Module.ProviderForLocalConfig(rc.ProviderConfigAddr()),
		self:     e.body(body, scope),
		planned:  planned,
		lookup:   lookup,
	}
}

// keyPath returns the origins of the record at a key path whose planned
// value has n leaves, and the reasons for the leaves without an origin.
// countable is false if the number of planned leaves isn't known.
func (r *originRecord) keyPath(kp []string, n int, countable bool) ([]*proto.Origin, []proto.NoOriginReason) {
	if policyIgnoresChanges(r.rc, kp[0]) {
		return nil, []proto.NoOriginReason{proto.NoOriginReason_IGNORE_CHANGES_NO_ORIGIN_REASON}
	}
	if !policyBodySets(r.body, kp[0]) {
		return nil, []proto.NoOriginReason{proto.NoOriginReason_NOT_CONFIGURED_NO_ORIGIN_REASON}
	}
	leaves, reason, ok := r.e.leaves(r.self, r.planned.Type(), kp)
	if !ok {
		return nil, []proto.NoOriginReason{reason}
	}
	if !countable {
		return nil, []proto.NoOriginReason{proto.NoOriginReason_PROVIDER_CHANGED_NO_ORIGIN_REASON}
	}

	// Leaves that are known to be null contribute no planned leaves, unless
	// the provider planned a value for them. If the numbers match neither
	// way, the planned value doesn't fit the configuration.
	knownNull := make([]bool, len(leaves))
	k := 0
	for i, leaf := range leaves {
		if r.knownNull(leaf, kp) {
			knownNull[i] = true
			k++
		}
	}
	allPlanned := n == len(leaves)
	if !allPlanned && n != len(leaves)-k {
		return nil, []proto.NoOriginReason{proto.NoOriginReason_PROVIDER_CHANGED_NO_ORIGIN_REASON}
	}

	seen := make(map[string]bool)
	reasons := make(map[proto.NoOriginReason]bool)
	var origins []*proto.Origin
	for i, leaf := range leaves {
		if knownNull[i] && !allPlanned {
			continue
		}
		switch sym := leaf.sym.(type) {
		case symRef:
			if len(sym.path) == 0 || !r.e.originAllowed(sym, leaf.ty, r.lookup) {
				reasons[proto.NoOriginReason_UNSUPPORTED_NO_ORIGIN_REASON] = true
				continue
			}
			id := sym.addr.String() + "#" + strings.Join(sym.path, ".")
			if seen[id] {
				continue
			}
			seen[id] = true
			origins = append(origins, &proto.Origin{Address: sym.addr.String(), Path: policyAttrPath(sym.path)})
		case symLit:
			if sym.val.IsNull() {
				reasons[proto.NoOriginReason_NOT_CONFIGURED_NO_ORIGIN_REASON] = true
			} else {
				reasons[proto.NoOriginReason_LITERAL_NO_ORIGIN_REASON] = true
			}
		default:
			reasons[policyStructureReason(sym)] = true
		}
	}
	sort.Slice(origins, func(i, j int) bool {
		if origins[i].Address != origins[j].Address {
			return origins[i].Address < origins[j].Address
		}
		return strings.Join(policyKeyPathNames(origins[i].Path), ".") < strings.Join(policyKeyPathNames(origins[j].Path), ".")
	})
	ret := make([]proto.NoOriginReason, 0, len(reasons))
	for reason := range reasons {
		ret = append(ret, reason)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return origins, ret
}

// knownNull returns true if a leaf of the record at key path kp is known to
// be null: a null literal, a reference to a planned null value, or an unset
// attribute the provider doesn't compute.
func (r *originRecord) knownNull(leaf originLeaf, kp []string) bool {
	switch sym := leaf.sym.(type) {
	case symLit:
		return sym.val.IsNull()
	case symRef:
		if len(sym.path) == 0 {
			return false
		}
		val, ok := r.lookup.plannedValue(sym.addr, sym.path)
		return ok && val.IsNull()
	default:
		if !policySymUnset(sym) {
			return false
		}
		computed, ok := r.lookup.attrComputed(r.provider, r.rc.Type, kp)
		return ok && !computed
	}
}

// policyBodySets returns true if a resource body sets an argument or has
// blocks, including dynamic ones, of the given name.
func policyBodySets(body *hclsyntax.Body, name string) bool {
	if _, ok := body.Attributes[name]; ok {
		return true
	}
	for _, block := range body.Blocks {
		if block.Type == name || (block.Type == "dynamic" && len(block.Labels) > 0 && block.Labels[0] == name) {
			return true
		}
	}
	return false
}

// policyIgnoresChanges returns true if the resource's ignore_changes covers
// the attribute name, or might cover it.
func policyIgnoresChanges(rc *configs.Resource, name string) bool {
	if rc.Managed == nil {
		return false
	}
	if rc.Managed.IgnoreAllChanges {
		return true
	}
	for _, trav := range rc.Managed.IgnoreChanges {
		if len(trav) == 0 {
			continue
		}
		switch step := trav[0].(type) {
		case hcl.TraverseAttr:
			if step.Name == name {
				return true
			}
		case hcl.TraverseRoot:
			if step.Name == name {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// policyPlannedLeaves returns the number of leaves of a planned value at a
// key path, flattening lists, sets and tuples at any position like the policy
// engine does. A null value contributes no leaves and an unknown primitive
// value is one leaf. ok is false if the number can't be known: a wholly
// unknown collection or object, or a value that doesn't fit the key path.
func policyPlannedLeaves(v cty.Value, steps []string) (n int, ok bool) {
	if v.IsNull() {
		return 0, true
	}
	ty := v.Type()
	if !v.IsKnown() {
		if len(steps) == 0 && ty.IsPrimitiveType() {
			return 1, true
		}
		return 0, false
	}
	switch {
	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		for it := v.ElementIterator(); it.Next(); {
			_, el := it.Element()
			c, ok := policyPlannedLeaves(el, steps)
			if !ok {
				return 0, false
			}
			n += c
		}
		return n, true
	case len(steps) > 0:
		if !ty.IsObjectType() || !ty.HasAttribute(steps[0]) {
			return 0, false
		}
		return policyPlannedLeaves(v.GetAttr(steps[0]), steps[1:])
	case ty.IsPrimitiveType():
		return 1, true
	default:
		return 0, false
	}
}

// Symbolic values. They describe what an expression evaluates to in terms of
// other objects, as far as the evaluation preserves values.
type originSym interface{}

// symOpaque is a value that has no origin, or whose structure isn't known.
// reason says why; the zero reason means UNSUPPORTED.
type symOpaque struct{ reason proto.NoOriginReason }

var (
	opaqueUnsupported   = symOpaque{reason: proto.NoOriginReason_UNSUPPORTED_NO_ORIGIN_REASON}
	opaqueExpression    = symOpaque{reason: proto.NoOriginReason_EXPRESSION_NO_ORIGIN_REASON}
	opaqueDataSource    = symOpaque{reason: proto.NoOriginReason_DATA_SOURCE_NO_ORIGIN_REASON}
	opaqueVariable      = symOpaque{reason: proto.NoOriginReason_VARIABLE_NO_ORIGIN_REASON}
	opaqueNotConfigured = symOpaque{reason: proto.NoOriginReason_NOT_CONFIGURED_NO_ORIGIN_REASON}
)

func (o symOpaque) why() proto.NoOriginReason {
	if o.reason == proto.NoOriginReason_INVALID_NO_ORIGIN_REASON {
		return proto.NoOriginReason_UNSUPPORTED_NO_ORIGIN_REASON
	}
	return o.reason
}

// symLit is a literal value, or count.index or each.key.
type symLit struct{ val cty.Value }

// symRef is the value of a managed resource instance, or of one of its
// attributes when path isn't empty.
type symRef struct {
	addr addrs.AbsResourceInstance
	path []string
}

// symObj is an object whose attributes are evaluated on demand: an object
// constructor, a block body, or a module instance's outputs.
type symObj struct{ attr func(name string) originSym }

// symSeq is a sequence: a tuple constructor, a splat result, or the blocks of
// one type in a body.
type symSeq struct {
	elems  []originSym
	blocks bool
}

// symColl is the instances of a resource or module call that has count or
// for_each.
type symColl struct {
	keyType addrs.InstanceKeyType
	elems   map[addrs.InstanceKey]originSym
}

type originLeaf struct {
	sym originSym
	ty  cty.Type
}

// originMaxDepth bounds the evaluation depth, as a safety net.
const originMaxDepth = 100

type originEval struct {
	cfg        *configs.Config
	exp        *instances.Expander
	overridden func(addrs.ModuleInstance) bool
	depth      int
}

func (e *originEval) moduleOverridden(mod addrs.ModuleInstance) bool {
	return e.overridden != nil && e.overridden(mod)
}

type originScope struct {
	mod  addrs.ModuleInstance
	cfg  *configs.Config
	rep  *originRepetition
	anon map[*hclsyntax.AnonSymbolExpr]originSym
}

// originRepetition is the repetition data of the object whose expressions
// are evaluated.
type originRepetition struct {
	key addrs.InstanceKey
	// each returns each.value; nil without for_each.
	each func() originSym
}

func (e *originEval) resourceRepetition(rc *configs.Resource, addr addrs.AbsResourceInstance) *originRepetition {
	switch addr.Resource.Key.(type) {
	case addrs.IntKey:
		return &originRepetition{key: addr.Resource.Key}
	case addrs.StringKey:
		return &originRepetition{key: addr.Resource.Key, each: func() originSym {
			if rc.ForEach == nil {
				return opaqueUnsupported
			}
			return e.index(e.eval(rc.ForEach, &originScope{mod: addr.Module, cfg: e.cfg.DescendantForInstance(addr.Module)}), instanceKeySym(addr.Resource.Key))
		}}
	default:
		return nil
	}
}

func instanceKeySym(key addrs.InstanceKey) originSym {
	if key == nil || key == addrs.NoKey {
		return opaqueUnsupported
	}
	return symLit{val: key.Value()}
}

// body returns the symbolic value of a resource body or nested block body.
func (e *originEval) body(body *hclsyntax.Body, s *originScope) originSym {
	return symObj{attr: func(name string) originSym {
		if attr, ok := body.Attributes[name]; ok {
			return e.eval(attr.Expr, s)
		}
		seq := symSeq{blocks: true}
		for _, block := range body.Blocks {
			if block.Type == "dynamic" && len(block.Labels) > 0 && block.Labels[0] == name {
				return opaqueUnsupported
			}
			if block.Type == name {
				seq.elems = append(seq.elems, e.body(block.Body, s))
			}
		}
		return seq
	}}
}

func (e *originEval) eval(expr hcl.Expression, s *originScope) originSym {
	if s == nil || s.cfg == nil || e.depth >= originMaxDepth {
		return opaqueUnsupported
	}
	e.depth++
	defer func() { e.depth-- }()

	switch expr := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		return symLit{val: expr.Val}
	case *hclsyntax.TemplateExpr:
		if len(expr.Parts) == 1 {
			if lit, ok := expr.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				return symLit{val: lit.Val}
			}
		}
		return opaqueExpression
	case *hclsyntax.TemplateWrapExpr:
		return e.eval(expr.Wrapped, s)
	case *hclsyntax.ParenthesesExpr:
		return e.eval(expr.Expression, s)
	case *hclsyntax.TupleConsExpr:
		seq := symSeq{}
		for _, el := range expr.Exprs {
			seq.elems = append(seq.elems, e.eval(el, s))
		}
		return seq
	case *hclsyntax.ObjectConsExpr:
		attrs := make(map[string]originSym, len(expr.Items))
		for _, item := range expr.Items {
			name, ok := objectConsKey(item.KeyExpr)
			if !ok {
				return opaqueExpression
			}
			attrs[name] = e.eval(item.ValueExpr, s)
		}
		return symObj{attr: func(name string) originSym {
			if v, ok := attrs[name]; ok {
				return v
			}
			return opaqueNotConfigured
		}}
	case *hclsyntax.ScopeTraversalExpr:
		return e.traversal(expr.Traversal, s)
	case *hclsyntax.RelativeTraversalExpr:
		return e.steps(e.eval(expr.Source, s), expr.Traversal)
	case *hclsyntax.IndexExpr:
		return e.index(e.eval(expr.Collection, s), e.eval(expr.Key, s))
	case *hclsyntax.SplatExpr:
		return e.splat(expr, s)
	case *hclsyntax.AnonSymbolExpr:
		if v, ok := s.anon[expr]; ok {
			return v
		}
		return opaqueUnsupported
	default:
		// Function calls, conditionals, for expressions, operators,
		// templates with several parts and anything else.
		return opaqueExpression
	}
}

// objectConsKey returns the literal key of an object constructor item.
func objectConsKey(expr hclsyntax.Expression) (string, bool) {
	key, ok := expr.(*hclsyntax.ObjectConsKeyExpr)
	if !ok {
		return "", false
	}
	if !key.ForceNonLiteral {
		if name := hcl.ExprAsKeyword(key.Wrapped); name != "" {
			return name, true
		}
	}
	switch wrapped := key.Wrapped.(type) {
	case *hclsyntax.TemplateExpr:
		if len(wrapped.Parts) == 1 {
			if lit, ok := wrapped.Parts[0].(*hclsyntax.LiteralValueExpr); ok && lit.Val.Type() == cty.String && lit.Val.IsKnown() && !lit.Val.IsNull() {
				return lit.Val.AsString(), true
			}
		}
	case *hclsyntax.LiteralValueExpr:
		if wrapped.Val.Type() == cty.String && wrapped.Val.IsKnown() && !wrapped.Val.IsNull() {
			return wrapped.Val.AsString(), true
		}
	}
	return "", false
}

func (e *originEval) traversal(trav hcl.Traversal, s *originScope) originSym {
	ref, diags := addrs.ParseRef(trav)
	if diags.HasErrors() || ref == nil {
		return opaqueUnsupported
	}
	var base originSym
	switch subj := ref.Subject.(type) {
	case addrs.Resource:
		base = e.resource(subj, s)
	case addrs.ResourceInstance:
		base = e.resourceInstance(subj, s)
	case addrs.LocalValue:
		local := s.cfg.Module.Locals[subj.Name]
		if local == nil {
			return opaqueUnsupported
		}
		base = e.eval(local.Expr, &originScope{mod: s.mod, cfg: s.cfg})
	case addrs.InputVariable:
		base = e.variable(subj.Name, s)
	case addrs.ModuleCall:
		base = e.moduleCall(subj, s)
	case addrs.ModuleCallInstance:
		base = e.moduleCallInstance(subj, s)
	case addrs.ModuleCallInstanceOutput:
		base = e.getAttr(e.moduleCallInstance(subj.Call, s), subj.Name)
	case addrs.CountAttr:
		if subj.Name != "index" || s.rep == nil {
			return opaqueUnsupported
		}
		if _, ok := s.rep.key.(addrs.IntKey); !ok {
			return opaqueUnsupported
		}
		base = instanceKeySym(s.rep.key)
	case addrs.ForEachAttr:
		if s.rep == nil || s.rep.each == nil {
			return opaqueUnsupported
		}
		switch subj.Name {
		case "key":
			base = instanceKeySym(s.rep.key)
		case "value":
			base = s.rep.each()
		default:
			return opaqueUnsupported
		}
	default:
		// path, terraform, self and others have no origin.
		return opaqueUnsupported
	}
	return e.steps(base, ref.Remaining)
}

func (e *originEval) steps(base originSym, trav hcl.Traversal) originSym {
	for _, step := range trav {
		switch step := step.(type) {
		case hcl.TraverseAttr:
			base = e.getAttr(base, step.Name)
		case hcl.TraverseIndex:
			base = e.index(base, symLit{val: step.Key})
		default:
			return opaqueUnsupported
		}
	}
	return base
}

func (e *originEval) getAttr(base originSym, name string) originSym {
	switch base := base.(type) {
	case symObj:
		return base.attr(name)
	case symRef:
		path := make([]string, len(base.path), len(base.path)+1)
		copy(path, base.path)
		return symRef{addr: base.addr, path: append(path, name)}
	case symOpaque:
		return base
	default:
		return opaqueUnsupported
	}
}

func (e *originEval) index(base originSym, key originSym) originSym {
	if base, ok := base.(symOpaque); ok {
		return base
	}
	lit, ok := key.(symLit)
	if !ok || !lit.val.IsWhollyKnown() || lit.val.IsNull() || lit.val.IsMarked() {
		return opaqueExpression
	}
	k, err := addrs.ParseInstanceKey(lit.val)
	if err != nil {
		return opaqueUnsupported
	}
	switch base := base.(type) {
	case symColl:
		if el, ok := base.elems[k]; ok {
			return el
		}
	case symSeq:
		if i, ok := k.(addrs.IntKey); ok && int(i) >= 0 && int(i) < len(base.elems) {
			return base.elems[i]
		}
	case symObj:
		if name, ok := k.(addrs.StringKey); ok {
			return base.attr(string(name))
		}
	}
	return opaqueUnsupported
}

func (e *originEval) splat(expr *hclsyntax.SplatExpr, s *originScope) originSym {
	var elems []originSym
	switch src := e.eval(expr.Source, s).(type) {
	case symSeq:
		elems = src.elems
	case symColl:
		if src.keyType != addrs.IntKeyType {
			return opaqueExpression
		}
		keys := make([]int, 0, len(src.elems))
		for k := range src.elems {
			keys = append(keys, int(k.(addrs.IntKey)))
		}
		sort.Ints(keys)
		for _, k := range keys {
			elems = append(elems, src.elems[addrs.IntKey(k)])
		}
	case symRef:
		// A splat over a single object wraps it in a tuple.
		if len(src.path) != 0 {
			return opaqueUnsupported
		}
		elems = []originSym{src}
	case symOpaque:
		return src
	default:
		return opaqueUnsupported
	}

	seq := symSeq{}
	for _, el := range elems {
		anon := make(map[*hclsyntax.AnonSymbolExpr]originSym, len(s.anon)+1)
		for k, v := range s.anon {
			anon[k] = v
		}
		anon[expr.Item] = el
		seq.elems = append(seq.elems, e.eval(expr.Each, &originScope{mod: s.mod, cfg: s.cfg, rep: s.rep, anon: anon}))
	}
	return seq
}

func (e *originEval) resource(res addrs.Resource, s *originScope) originSym {
	if opaque, ok := policyUnmanaged(res.Mode); ok {
		return opaque
	}
	if s.cfg.Module.ResourceByAddr(res) == nil {
		return opaqueUnsupported
	}
	abs := res.Absolute(s.mod)
	if !e.exp.ResourceInstanceExpanded(abs) {
		return opaqueUnsupported
	}
	keyType, keys, unknown := e.exp.ResourceInstanceKeys(abs)
	if unknown {
		return opaqueUnsupported
	}
	if keyType == addrs.NoKeyType {
		return symRef{addr: res.Instance(addrs.NoKey).Absolute(s.mod)}
	}
	coll := symColl{keyType: keyType, elems: make(map[addrs.InstanceKey]originSym, len(keys))}
	for _, k := range keys {
		coll.elems[k] = symRef{addr: res.Instance(k).Absolute(s.mod)}
	}
	return coll
}

func (e *originEval) resourceInstance(inst addrs.ResourceInstance, s *originScope) originSym {
	if opaque, ok := policyUnmanaged(inst.Resource.Mode); ok {
		return opaque
	}
	if s.cfg.Module.ResourceByAddr(inst.Resource) == nil {
		return opaqueUnsupported
	}
	abs := inst.Resource.Absolute(s.mod)
	if !e.exp.ResourceInstanceExpanded(abs) {
		return opaqueUnsupported
	}
	keyType, keys, _ := e.exp.ResourceInstanceKeys(abs)
	if !instanceKeyKnown(inst.Key, keyType, keys) {
		return opaqueUnsupported
	}
	return symRef{addr: inst.Absolute(s.mod)}
}

// policyUnmanaged returns the symbolic value of a reference to a resource
// that isn't managed. Data sources and ephemeral resources are compared by
// value.
func policyUnmanaged(mode addrs.ResourceMode) (symOpaque, bool) {
	switch mode {
	case addrs.ManagedResourceMode:
		return symOpaque{}, false
	case addrs.DataResourceMode, addrs.EphemeralResourceMode:
		return opaqueDataSource, true
	default:
		return opaqueUnsupported, true
	}
}

func policyInstanceKeyType(key addrs.InstanceKey) addrs.InstanceKeyType {
	switch key.(type) {
	case addrs.IntKey:
		return addrs.IntKeyType
	case addrs.StringKey:
		return addrs.StringKeyType
	default:
		return addrs.NoKeyType
	}
}

func instanceKeyKnown(key addrs.InstanceKey, keyType addrs.InstanceKeyType, keys []addrs.InstanceKey) bool {
	if key == nil {
		key = addrs.NoKey
	}
	if policyInstanceKeyType(key) != keyType {
		return false
	}
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

// variable follows a module input variable to the argument of the module
// call, in the parent module instance.
func (e *originEval) variable(name string, s *originScope) originSym {
	if s.mod.IsRoot() || s.cfg.Parent == nil {
		return opaqueVariable
	}
	parentMod, callInst := s.mod.CallInstance()
	parentCfg := s.cfg.Parent
	call := parentCfg.Module.ModuleCalls[callInst.Call.Name]
	if call == nil {
		return opaqueUnsupported
	}
	body, ok := call.Config.(*hclsyntax.Body)
	if !ok {
		return opaqueUnsupported
	}
	attr, ok := body.Attributes[name]
	if !ok {
		// The variable has its default value.
		return opaqueVariable
	}
	var rep *originRepetition
	switch callInst.Key.(type) {
	case addrs.IntKey:
		rep = &originRepetition{key: callInst.Key}
	case addrs.StringKey:
		rep = &originRepetition{key: callInst.Key, each: func() originSym {
			if call.ForEach == nil {
				return opaqueUnsupported
			}
			return e.index(e.eval(call.ForEach, &originScope{mod: parentMod, cfg: parentCfg}), instanceKeySym(callInst.Key))
		}}
	}
	return e.eval(attr.Expr, &originScope{mod: parentMod, cfg: parentCfg, rep: rep})
}

func (e *originEval) moduleCall(call addrs.ModuleCall, s *originScope) originSym {
	childCfg := s.cfg.Children[call.Name]
	abs := call.Absolute(s.mod)
	if childCfg == nil || !e.exp.AbsModuleCallExpanded(abs) {
		return opaqueUnsupported
	}
	keyType, keys, unknown := e.exp.GetModuleCallInstanceKeys(abs)
	if unknown {
		return opaqueUnsupported
	}
	if keyType == addrs.NoKeyType {
		return e.moduleInstance(s.mod.Child(call.Name, addrs.NoKey), childCfg)
	}
	coll := symColl{keyType: keyType, elems: make(map[addrs.InstanceKey]originSym, len(keys))}
	for _, k := range keys {
		coll.elems[k] = e.moduleInstance(s.mod.Child(call.Name, k), childCfg)
	}
	return coll
}

func (e *originEval) moduleCallInstance(inst addrs.ModuleCallInstance, s *originScope) originSym {
	childCfg := s.cfg.Children[inst.Call.Name]
	abs := inst.Call.Absolute(s.mod)
	if childCfg == nil || !e.exp.AbsModuleCallExpanded(abs) {
		return opaqueUnsupported
	}
	keyType, keys, _ := e.exp.GetModuleCallInstanceKeys(abs)
	if !instanceKeyKnown(inst.Key, keyType, keys) {
		return opaqueUnsupported
	}
	return e.moduleInstance(s.mod.Child(inst.Call.Name, inst.Key), childCfg)
}

// moduleInstance returns a module instance's outputs, which are evaluated
// in the module instance. moduleCall and moduleCallInstance return module
// instances through it, so this is where overridden modules are excluded.
func (e *originEval) moduleInstance(mod addrs.ModuleInstance, cfg *configs.Config) originSym {
	if e.moduleOverridden(mod) {
		return opaqueUnsupported
	}
	return symObj{attr: func(name string) originSym {
		out := cfg.Module.Outputs[name]
		if out == nil {
			return opaqueUnsupported
		}
		return e.eval(out.Expr, &originScope{mod: mod, cfg: cfg})
	}}
}

// leaves walks a symbolic value along a key path like policyPlannedLeaves
// walks the planned value of type ty, and returns the leaves with their
// types. ok is false if the leaves can't be known, because a collection
// position isn't syntactic, and reason says why.
func (e *originEval) leaves(s originSym, ty cty.Type, steps []string) ([]originLeaf, proto.NoOriginReason, bool) {
	switch {
	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		seq, ok := s.(symSeq)
		if !ok {
			return nil, policyStructureReason(s), false
		}
		elemTy := func(int) cty.Type { return ty.ElementType() }
		if ty.IsTupleType() {
			elemTys := ty.TupleElementTypes()
			if len(seq.elems) != len(elemTys) {
				return nil, proto.NoOriginReason_PROVIDER_CHANGED_NO_ORIGIN_REASON, false
			}
			elemTy = func(i int) cty.Type { return elemTys[i] }
		}
		var ret []originLeaf
		for i, el := range seq.elems {
			l, reason, ok := e.leaves(el, elemTy(i), steps)
			if !ok {
				return nil, reason, false
			}
			ret = append(ret, l...)
		}
		return ret, proto.NoOriginReason_INVALID_NO_ORIGIN_REASON, true
	case len(steps) > 0:
		if !ty.IsObjectType() || !ty.HasAttribute(steps[0]) {
			return nil, proto.NoOriginReason_PROVIDER_CHANGED_NO_ORIGIN_REASON, false
		}
		switch s := s.(type) {
		case symObj, symRef:
			return e.leaves(e.getAttr(s, steps[0]), ty.AttributeType(steps[0]), steps[1:])
		case symSeq:
			// A single nested block.
			if s.blocks && len(s.elems) == 1 {
				return e.leaves(s.elems[0], ty, steps)
			}
		}
		return nil, policyStructureReason(s), false
	case ty.IsPrimitiveType():
		return []originLeaf{{sym: s, ty: ty}}, proto.NoOriginReason_INVALID_NO_ORIGIN_REASON, true
	default:
		return nil, proto.NoOriginReason_UNSUPPORTED_NO_ORIGIN_REASON, false
	}
}

// policyStructureReason returns why a symbolic value has no origin, or
// doesn't have the structure of the planned value.
func policyStructureReason(s originSym) proto.NoOriginReason {
	switch s := s.(type) {
	case symOpaque:
		return s.why()
	case symLit:
		if s.val.IsNull() {
			return proto.NoOriginReason_NOT_CONFIGURED_NO_ORIGIN_REASON
		}
	}
	if policySymUnset(s) {
		return proto.NoOriginReason_NOT_CONFIGURED_NO_ORIGIN_REASON
	}
	return proto.NoOriginReason_UNSUPPORTED_NO_ORIGIN_REASON
}

// policySymUnset returns true if a symbolic value is an attribute that isn't
// set: a body without the argument or blocks, or an object constructor
// without the attribute.
func policySymUnset(s originSym) bool {
	switch s := s.(type) {
	case symSeq:
		return s.blocks && len(s.elems) == 0
	case symOpaque:
		return s.reason == proto.NoOriginReason_NOT_CONFIGURED_NO_ORIGIN_REASON
	default:
		return false
	}
}

// originAllowed checks the conditions of contract §5 for a single origin:
// the referenced attribute is a primitive attribute of the referenced
// managed resource type, of the same type as the record's leaf, and isn't
// known to be null.
func (e *originEval) originAllowed(ref symRef, leafTy cty.Type, lookup originLookup) bool {
	if e.moduleOverridden(ref.addr.Module) {
		return false
	}
	modCfg := e.cfg.DescendantForInstance(ref.addr.Module)
	if modCfg == nil {
		return false
	}
	rc := modCfg.Module.ResourceByAddr(ref.addr.Resource.Resource)
	if rc == nil || rc.Mode != addrs.ManagedResourceMode {
		return false
	}
	provider := modCfg.Module.ProviderForLocalConfig(rc.ProviderConfigAddr())
	ty, ok := lookup.attrType(provider, rc.Type, ref.path)
	if !ok || !ty.Equals(leafTy) {
		return false
	}
	if val, ok := lookup.plannedValue(ref.addr, ref.path); ok && val.IsNull() {
		return false
	}
	return true
}
