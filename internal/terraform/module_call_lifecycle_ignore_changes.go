// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

func moduleCallExtraIgnoreTraversals(root *configs.Config, resInst addrs.AbsResourceInstance) ([]hcl.Traversal, bool) {
	var extra []hcl.Traversal
	ignoreAll := false

	mi := resInst.Module // full module *instance* path (includes keys)
	if len(mi) == 0 {
		return nil, false
	}

	// For each step in the module instance path, look at the corresponding
	// module call in the parent module config.
	for depth := 1; depth <= len(mi); depth++ {
		parentMI := mi[:depth-1]
		step := mi[depth-1] // ModuleInstanceStep{Name, InstanceKey}

		parentCfg := root.Descendant(parentMI.Module()) // config tree is keyed by static module path
		if parentCfg == nil || parentCfg.Module == nil {
			continue
		}

		mc := parentCfg.Module.ModuleCalls[step.Name]
		if mc == nil || mc.Managed == nil {
			continue
		}

		// "ignore_changes = all" on the module call lifecycle:
		// treat as ignore-all for every managed resource under it.
		if mc.Managed.IgnoreAllChanges {
			ignoreAll = true
		}

		for _, t := range mc.Managed.IgnoreChanges {
			if rem, ok := matchModuleIgnoreTraversal(t, step.InstanceKey, resInst.Resource); ok {
				extra = append(extra, rem)
			}
		}
	}

	return extra, ignoreAll
}

// matchModuleIgnoreTraversal checks whether a module-call ignore traversal
// (like self[0].google_storage_bucket_iam_binding.authoritative["k"].members)
// applies to this resource instance. If so it returns the attribute traversal
// relative to the resource instance (e.g. members, labels["x"], etc).
func matchModuleIgnoreTraversal(
	t hcl.Traversal,
	moduleKey addrs.InstanceKey,
	target addrs.ResourceInstance,
) (hcl.Traversal, bool) {

	if len(t) < 2 {
		return nil, false
	}

	// module_call.go stores ignore_changes items using hcl.RelTraversalForExpr,
	// which can yield TraverseAttr for the first step.
	var firstName string
	switch s := t[0].(type) {
	case hcl.TraverseRoot:
		firstName = s.Name
	case hcl.TraverseAttr:
		firstName = s.Name
	default:
		return nil, false
	}
	if firstName != "self" {
		return nil, false
	}

	i := 1

	// Optional self[<key>] filter on module instance key.
	if i < len(t) {
		if idx, ok := t[i].(hcl.TraverseIndex); ok {
			k, err := addrs.ParseInstanceKey(idx.Key)
			if err != nil {
				return nil, false
			}
			if moduleKey == addrs.NoKey || k != moduleKey {
				return nil, false
			}
			i++
		}
	}

	rest := t[i:]
	if len(rest) < 3 {
		// Need at least <type>.<name>.<attr...>
		return nil, false
	}

	// Normalize first step to TraverseRoot so ParseRef is happy in all cases.
	if a, ok := rest[0].(hcl.TraverseAttr); ok {
		rest = append(hcl.Traversal{
			hcl.TraverseRoot{Name: a.Name, SrcRange: a.SrcRange},
		}, rest[1:]...)
	}

	ref, diags := addrs.ParseRef(rest)
	if diags.HasErrors() || ref == nil {
		return nil, false
	}

	subj, ok := ref.Subject.(addrs.ResourceInstance)
	if !ok {
		return nil, false
	}

	// Only managed resources.
	if subj.Resource.Mode != addrs.ManagedResourceMode {
		return nil, false
	}

	// Match resource type+name.
	if subj.Resource != target.Resource {
		return nil, false
	}

	// If the ignore rule specifies a resource instance key, require it to match
	// only when the target instance key is known. If target.Key is NoKey, we
	// can't reliably filter to one instance at this point, so we treat it as
	// applying to the resource block (and thus all instances).
	if subj.Key != addrs.NoKey && target.Key != addrs.NoKey && subj.Key != target.Key {
		return nil, false
	}

	if len(ref.Remaining) == 0 {
		// No attribute path => nothing to ignore
		return nil, false
	}

	return ref.Remaining, true
}

func copyResourceForIgnoreAppend(r *configs.Resource) *configs.Resource {
	rc := *r

	if r.Managed != nil {
		mc := *r.Managed
		if len(mc.IgnoreChanges) != 0 {
			cp := make([]hcl.Traversal, len(mc.IgnoreChanges))
			copy(cp, mc.IgnoreChanges)
			mc.IgnoreChanges = cp
		}
		rc.Managed = &mc
	}

	return &rc
}
