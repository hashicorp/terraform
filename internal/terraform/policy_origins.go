// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/schemarepo"
)

// originMaxDepth bounds the references that resolving one origin follows.
const originMaxDepth = 100

// policyOriginResolver resolves the origins of the key paths of a plan's
// managed resource instances (contract §3.5): the resource attributes that
// their configured values are copied from, by plain references only.
type policyOriginResolver struct {
	cfg     *configs.Config
	schemas *schemarepo.Schemas
	// overridden reports whether a module instance is overridden by the
	// walk's test overrides; nil without overrides. The outputs of an
	// overridden module come from the override instead of its
	// configuration, so nothing in it is followed.
	overridden func(addrs.ModuleInstance) bool
}

// policyOverriddenModules returns whether module instances are overridden
// by the given test overrides, or nil if there are no overrides. Overridden
// resources don't need excluding from origins: overrides only fill in their
// computed attributes, so their configured attributes still come from their
// configuration.
func policyOverriddenModules(overrides *mocking.Overrides) func(addrs.ModuleInstance) bool {
	if overrides.Empty() {
		return nil
	}
	return overrides.IsOverridden
}

// originsFor returns the origins of the planned value of a managed resource
// instance at the given key paths, by dot-joined key path. Key paths without
// origins have no entry. A key path has origins only if it reads one
// primitive value, through attributes and single nested blocks, and its
// attribute's expression in the configuration is a plain reference.
//
// Every origin of a key path is dropped when the path's first step is in
// ignore_changes and the change is an update or a no-op, which keep the prior
// value; when the planned value at the path is null; and when the planned
// value isn't the configured one, which is configured, because then the
// provider changed it.
func (o *policyOriginResolver) originsFor(addr addrs.AbsResourceInstance, action plans.Action, planned, configured cty.Value, keyPaths [][]string) map[string][]string {
	if o.cfg == nil || planned == cty.NilVal || configured == cty.NilVal || len(keyPaths) == 0 {
		return nil
	}
	if o.overridden != nil && o.overridden(addr.Module) {
		return nil
	}
	modCfg := o.cfg.DescendantForInstance(addr.Module)
	if modCfg == nil {
		return nil
	}
	rc := modCfg.Module.ResourceByAddr(addr.Resource.Resource)
	if rc == nil || rc.Mode != addrs.ManagedResourceMode {
		return nil
	}
	// JSON bodies and bodies merged from override files are other types.
	body, ok := rc.Config.(*hclsyntax.Body)
	if !ok {
		return nil
	}
	planned, _ = planned.UnmarkDeep()
	configured, _ = configured.UnmarkDeep()

	var ret map[string][]string
	for _, kp := range keyPaths {
		if len(kp) == 0 || !policyScalarPath(planned.Type(), kp) {
			continue
		}
		if policyIgnoreChangesApplies(action) && policyIgnoresChanges(rc, kp[0]) {
			continue
		}
		p, c := policyPathValue(planned, kp), policyPathValue(configured, kp)
		if p == cty.NilVal || c == cty.NilVal || (p.IsKnown() && p.IsNull()) || !policySameValue(c, p) {
			continue
		}
		expr := policyPathExpr(body, kp)
		if expr == nil {
			continue
		}
		if origin, ok := o.resolve(expr, addr.Module, 0); ok {
			if ret == nil {
				ret = make(map[string][]string)
			}
			ret[strings.Join(kp, ".")] = []string{origin}
		}
	}
	return ret
}

// policyPathValue returns the value at a path of attribute steps, which is
// null below a null and unknown below an unknown, or cty.NilVal if the path
// doesn't fit the value.
func policyPathValue(v cty.Value, path []string) cty.Value {
	for _, name := range path {
		ty := v.Type()
		if !ty.IsObjectType() || !ty.HasAttribute(name) {
			return cty.NilVal
		}
		if !v.IsKnown() {
			return cty.UnknownVal(ty.AttributeType(name))
		}
		if v.IsNull() {
			return cty.NullVal(ty.AttributeType(name))
		}
		v = v.GetAttr(name)
	}
	return v
}

// policySameValue returns true if a planned value keeps a configured one:
// both are known and equal, or both are unknown. A known planned value for
// an unknown configured one, or the reverse, isn't the same value as far as
// Terraform can tell.
func policySameValue(c, p cty.Value) bool {
	if !c.IsKnown() || !p.IsKnown() {
		return !c.IsKnown() && !p.IsKnown()
	}
	return c.Type().Equals(p.Type()) && c.Equals(p).True()
}

// policyPathExpr returns the expression of the attribute at a key path in a
// resource body, through blocks of which the body has exactly one, or nil if
// there's no such attribute. A key path that steps into an attribute has no
// expression: only plain references are followed, not object constructors.
func policyPathExpr(body *hclsyntax.Body, kp []string) hclsyntax.Expression {
	for _, name := range kp[:len(kp)-1] {
		var found *hclsyntax.Block
		for _, block := range body.Blocks {
			if block.Type == name || (block.Type == "dynamic" && len(block.Labels) > 0 && block.Labels[0] == name) {
				if found != nil || block.Type == "dynamic" {
					return nil
				}
				found = block
			}
		}
		if found == nil {
			return nil
		}
		body = found.Body
	}
	attr, ok := body.Attributes[kp[len(kp)-1]]
	if !ok {
		return nil
	}
	return attr.Expr
}

// resolve returns the origin of an expression in a module instance:
// references to resource attributes are the origin; references to local
// values, to input variables of a non-root module and to module outputs are
// followed to their expressions. Any other expression, including anything
// that isn't only a traversal, has no origin.
func (o *policyOriginResolver) resolve(expr hcl.Expression, mod addrs.ModuleInstance, depth int) (string, bool) {
	if depth > originMaxDepth {
		return "", false
	}
	trav, ok := expr.(*hclsyntax.ScopeTraversalExpr)
	if !ok {
		return "", false
	}
	ref, diags := addrs.ParseRef(trav.Traversal)
	if diags.HasErrors() {
		return "", false
	}
	modCfg := o.cfg.DescendantForInstance(mod)
	if modCfg == nil {
		return "", false
	}

	switch s := ref.Subject.(type) {
	case addrs.Resource:
		return o.resourceOrigin(modCfg, s.Absolute(mod).Instance(addrs.NoKey), ref.Remaining)
	case addrs.ResourceInstance:
		return o.resourceOrigin(modCfg, s.Absolute(mod), ref.Remaining)
	case addrs.LocalValue:
		local := modCfg.Module.Locals[s.Name]
		if local == nil || len(ref.Remaining) > 0 {
			return "", false
		}
		return o.resolve(local.Expr, mod, depth+1)
	case addrs.InputVariable:
		if mod.IsRoot() || len(ref.Remaining) > 0 {
			return "", false
		}
		parent, call := mod.CallInstance()
		parentCfg := o.cfg.DescendantForInstance(parent)
		if parentCfg == nil {
			return "", false
		}
		mc := parentCfg.Module.ModuleCalls[call.Call.Name]
		if mc == nil {
			return "", false
		}
		body, ok := mc.Config.(*hclsyntax.Body)
		if !ok {
			return "", false
		}
		attr, ok := body.Attributes[s.Name]
		if !ok {
			return "", false
		}
		return o.resolve(attr.Expr, parent, depth+1)
	case addrs.ModuleCallInstanceOutput:
		if len(ref.Remaining) > 0 {
			return "", false
		}
		mc := modCfg.Module.ModuleCalls[s.Call.Call.Name]
		if mc == nil || !policyKeyFits(s.Call.Key, mc.Count, mc.ForEach) {
			return "", false
		}
		child := mod.Child(s.Call.Call.Name, s.Call.Key)
		if o.overridden != nil && o.overridden(child) {
			return "", false
		}
		childCfg := o.cfg.DescendantForInstance(child)
		if childCfg == nil {
			return "", false
		}
		out := childCfg.Module.Outputs[s.Name]
		if out == nil {
			return "", false
		}
		return o.resolve(out.Expr, child, depth+1)
	default:
		return "", false
	}
}

// resourceOrigin returns the origin that a reference to a resource
// instance's attribute is: the attribute, if it's a primitive, non-write-only
// attribute of a managed resource or a data source of the configuration.
func (o *policyOriginResolver) resourceOrigin(modCfg *configs.Config, inst addrs.AbsResourceInstance, rest hcl.Traversal) (string, bool) {
	res := inst.Resource.Resource
	if res.Mode != addrs.ManagedResourceMode && res.Mode != addrs.DataResourceMode {
		return "", false
	}
	rc := modCfg.Module.ResourceByAddr(res)
	if rc == nil || !policyKeyFits(inst.Resource.Key, rc.Count, rc.ForEach) || len(rest) == 0 {
		return "", false
	}
	names := make([]string, 0, len(rest))
	path := make(cty.Path, 0, len(rest))
	for _, step := range rest {
		attr, ok := step.(hcl.TraverseAttr)
		if !ok {
			return "", false
		}
		names = append(names, attr.Name)
		path = path.GetAttr(attr.Name)
	}

	provider := modCfg.Module.ProviderForLocalConfig(rc.ProviderConfigAddr())
	schema := o.schemas.ResourceTypeConfig(provider, res.Mode, res.Type)
	if schema.Body == nil || !policyScalarPath(schema.Body.ImpliedType(), names) {
		return "", false
	}
	if attr := schema.Body.AttributeByPath(path); attr == nil || attr.WriteOnly {
		return "", false
	}
	return inst.String() + "." + strings.Join(names, "."), true
}

// policyKeyFits returns true if an instance key is a literal key of a
// resource or module call with the given count and for_each.
func policyKeyFits(key addrs.InstanceKey, count, forEach hcl.Expression) bool {
	switch {
	case count != nil:
		_, ok := key.(addrs.IntKey)
		return ok
	case forEach != nil:
		_, ok := key.(addrs.StringKey)
		return ok
	default:
		return key == addrs.NoKey
	}
}

// policyIgnoreChangesApplies returns true if ignore_changes applies to the
// planned value of a change with the given action: an update or a no-op of an
// existing object keeps the prior values of the ignored attributes. A create
// has no prior object, and a replace plans the new object from the whole
// configuration.
func policyIgnoreChangesApplies(action plans.Action) bool {
	return action == plans.Update || action == plans.NoOp
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
