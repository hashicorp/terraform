// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"slices"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/ctyutil"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/hashicorp/terraform/internal/lang/globalref"
)

// recordOrigin stores a managed attribute's immediate source in the same module.
// Data source configuration does not determine its returned values.
func (s *policyWalkState) recordOrigin(from, to globalref.ResourceAttr) {
	if s == nil || from.Resource.Resource.Resource.Mode != addrs.ManagedResourceMode || !from.Resource.Module.Equal(to.Resource.Module) {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.origins != nil {
		s.origins[policyAttributeKey(from)] = to
	}
}

func (s *policyWalkState) origin(attr globalref.ResourceAttr) (globalref.ResourceAttr, bool) {
	if s == nil {
		return globalref.ResourceAttr{}, false
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	key := policyAttributeKey(attr)
	origin, ok := s.origins[key]
	if !ok || policyAttributeKey(origin) == key {
		return globalref.ResourceAttr{}, false
	}
	return origin, true
}

// shareOrigin compares each endpoint and its immediate source. It never follows
// resource chains. A disabled origin map also disables self-identity proofs.
func (s *policyWalkState) shareOrigin(a, b globalref.ResourceAttr) bool {
	if s == nil || s.origins == nil {
		return false
	}
	aOrigins := []globalref.ResourceAttr{a}
	if origin, ok := s.origin(a); ok {
		aOrigins = append(aOrigins, origin)
	}
	bOrigins := []globalref.ResourceAttr{b}
	if origin, ok := s.origin(b); ok {
		bOrigins = append(bOrigins, origin)
	}
	for _, aOrigin := range aOrigins {
		for _, bOrigin := range bOrigins {
			if aOrigin.Resource.Equal(bOrigin.Resource) && ctyutil.NormalizePath(aOrigin.Attr).Equals(ctyutil.NormalizePath(bOrigin.Attr)) {
				return true
			}
		}
	}
	return false
}

func policyAttributeKey(attr globalref.ResourceAttr) string {
	return attr.Resource.String() + format.CtyPath(ctyutil.NormalizePath(attr.Attr))
}

// recordPolicyOrigins records eligible immediate origins only for the
// given canonical comparison paths from policy.Config.ComparisonPaths. Relationship
// validation must already have established that these select primitive values
// in the resource's schema.
func recordPolicyOrigins(ctx EvalContext, addr addrs.AbsResourceInstance, config *configs.Resource, schema *configschema.Block, paths []cty.Path) {
	if len(paths) == 0 || schema == nil || config == nil || config.Managed == nil || config.Managed.IgnoreAllChanges {
		return
	}
	module := ctx.Config().DescendantForInstance(addr.Module)
	if module == nil {
		return
	}
	policyState := ctx.PolicyState()
	if policyState == nil || policyState.origins == nil {
		return
	}

	var record func(cty.Path, hcl.Expression)
	record = func(current cty.Path, expr hcl.Expression) {
		normalized := ctyutil.NormalizePath(current)
		var exact, ancestor bool
		for _, path := range paths {
			exact = exact || path.Equals(normalized)
			ancestor = ancestor || path.HasPrefix(normalized)
		}

		// if this does not match any comparison path, then no need to record references
		if !ancestor {
			return
		}
		if traversal, diags := hcl.AbsTraversalForExpr(expr); !diags.HasErrors() {
			if !exact || isIgnoredPath(current, config) {
				return
			}

			// we walk the path and ensure that every schema metadata along the path
			// can be valid destinations.
			_, err := schema.WalkPath(current, func(_ cty.Path, attr *configschema.Attribute, _ *configschema.NestedBlock) error {
				if attr != nil && (attr.WriteOnly || attr.Computed) {
					return current.NewErrorf("computed and write-only attributes cannot be origin destinations")
				}
				return nil
			})
			if err != nil {
				return
			}

			source, ok := resolvePolicyResourceAttr(module.Module, addr.Module, traversal)
			if !ok {
				return
			}
			// TODO: Establish source-schema/type eligibility before claiming an
			// exact copy.
			policyState.recordOrigin(globalref.ResourceAttr{Resource: addr, Attr: current}, source)
			return
		}

		// A constructor can contain eligible members even though the whole
		// expression is not. Unknown keys make selection ambiguous.
		objExpr, ok := expr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			return
		}
		members := make(map[string]hcl.Expression)
		for _, item := range objExpr.ExprMap() {
			key, diags := item.Key.Value(nil)
			if diags.HasErrors() || key.Type() != cty.String || !key.IsKnown() || key.IsNull() {
				return
			}
			key, _ = key.Unmark()
			members[key.AsString()] = item.Value // Last key wins, as in HCL evaluation.
		}
		for name, member := range members {
			record(current.Copy().GetAttr(name), member)
		}
	}
	lang.VisitAttributes(config.Config, schema, func(path cty.Path, attr *hcl.Attribute) {
		record(path, attr.Expr)
	})
}

func isIgnoredPath(path cty.Path, config *configs.Resource) bool {
	ignoredPaths := make([]cty.Path, 0, len(config.Managed.IgnoreChanges))
	for _, traversal := range config.Managed.IgnoreChanges {
		path, _ := traversalToPath(traversal)
		ignoredPaths = append(ignoredPaths, ctyutil.NormalizePath(path))
	}

	path = ctyutil.NormalizePath(path)
	for _, ignored := range ignoredPaths {
		if path.HasPrefix(ignored) || ignored.HasPrefix(path) {
			return true
		}
	}
	return false
}

// resolvePolicyResourceAttr follows traversals up until resource attributes. Plain
// local aliases are also followed until they lead to resource attributes.
func resolvePolicyResourceAttr(module *configs.Module, moduleAddr addrs.ModuleInstance, traversal hcl.Traversal) (globalref.ResourceAttr, bool) {
	seen := make(map[string]bool)
	for {
		ref, diags := addrs.ParseRef(traversal)
		if diags.HasErrors() || ref == nil {
			return globalref.ResourceAttr{}, false
		}
		if attr, ok := (globalref.Reference{ContainerAddr: moduleAddr, LocalRef: ref}).ResourceAttr(); ok {
			return attr, len(attr.Attr) > 0
		}
		local, ok := ref.Subject.(addrs.LocalValue)
		if !ok || module == nil || seen[local.Name] {
			return globalref.ResourceAttr{}, false
		}
		seen[local.Name] = true
		localConfig := module.Locals[local.Name]
		alias, hclDiags := hcl.AbsTraversalForExpr(localConfig.Expr)
		if hclDiags.HasErrors() {
			return globalref.ResourceAttr{}, false
		}

		traversal = append(slices.Clone(alias), ref.Remaining...)
	}
}
