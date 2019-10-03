package projectconfigs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// staticValidate performs global static validation of a loaded configuration,
// catching early any problems that can be detected without any dynamic
// information such as context values, environment variables, etc.
//
// A config that passes static validation may still produce errors during
// validation, but we'd like to catch as many errors as possible here so that
// users can feel confident that in most cases once a project configuration
// has been accepted by one command it will be considered valid for other
// commands too.
func (c *Config) staticValidate() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	moreDiags := c.staticValidateAllReferences()
	diags = diags.Append(moreDiags)
	return diags
}

func (c *Config) staticValidateAllReferences() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// dynamicLocals tracks which locals are derived from the outputs of
	// workspaces either in this project or upstream, since it's illegal to
	// refer to those in certain contexts where we need to derive a value
	// prior to fetching any workspace states/outputs.
	dynamicLocals := map[string]*addrs.ProjectConfigReference{}
	localRefs := map[string]map[string]struct{}{}
	isDynamicAddr := func(addr addrs.ProjectReferenceable) bool {
		switch addr := addr.(type) {
		case addrs.ProjectWorkspace, addrs.ProjectUpstreamWorkspace:
			return true
		case addrs.LocalValue:
			return dynamicLocals[addr.Name] != nil
		default:
			return false
		}
	}

	for name, lv := range c.Locals {
		refs, moreDiags := c.getValidExprReferences(lv.Value)
		diags = diags.Append(moreDiags)
		dynamicLocals[name] = nil
		for _, ref := range refs {
			switch addr := ref.Subject.(type) {
			case addrs.ProjectWorkspace, addrs.ProjectUpstreamWorkspace:
				dynamicLocals[name] = ref
			case addrs.LocalValue:
				if _, ok := localRefs[name]; !ok {
					localRefs[name] = map[string]struct{}{}
				}
				localRefs[name][addr.Name] = struct{}{}
			}
		}
	}

	// "dynamicness" of a local value is transitive: if a local value refers
	// to another local value that is dynamic, then the referrer is also
	// dynamic. We'll propagate this by iterating until we stop making changes
	// and thus have converged on the final answer. This is guaranteed to
	// converge because there's a finite number of defined local values and
	// in the worst case we will mark all of them as dynamic and then stop.
	for {
		changed := false
	Outer:
		for name, dynamicRef := range dynamicLocals {
			if dynamicRef != nil {
				// If it's already dynamic then there's nothing to do here.
				continue
			}
			for otherName := range localRefs[name] {
				if dynamicLocals[otherName] != nil {
					dynamicLocals[name] = dynamicLocals[otherName]
					changed = true
					continue Outer
				}
			}
		}
		if !changed {
			break
		}
	}

	// We can now rely on isDynamicAddr to give accurate results for all
	// valid addresses.
	// TODO: Specialized error messages for dynamic local vs.
	// direct references to workspaces that gives the user a
	// stronger hint as to where the problem is.

	for _, ws := range c.Workspaces {
		forEachRefs, moreDiags := c.getValidExprReferences(ws.ForEach)
		diags = diags.Append(moreDiags)
		for _, ref := range forEachRefs {
			if isDynamicAddr(ref.Subject) {
				diags = diags.Append(hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid for_each reference",
					Detail:   fmt.Sprintf("Terraform must be able to determine the full set of workspaces prior to fetching any workspace outputs, so a for_each expression cannot be derived from a workspace or upstream output value."),
				})
			}
		}
		_, moreDiags = c.getValidExprReferences(ws.Variables)
		diags = diags.Append(moreDiags)
		_, moreDiags = c.getValidExprReferences(ws.ConfigSource)
		diags = diags.Append(moreDiags)
		_, moreDiags = c.getValidExprReferences(ws.Remote)
		diags = diags.Append(moreDiags)

		// NOTE: We can't validate the state_storage body yet because we
		// don't know it's schema here. That might contain invalid
		// references that we won't catch until later evaluation.
	}

	for _, us := range c.Upstreams {
		_, moreDiags := c.getValidExprReferences(us.Remote)
		diags = diags.Append(moreDiags)
	}

	// TODO: We should also check here for reference cycles in general. For
	// prototyping purposes we don't, which means downstream misbehavior is
	// likely if reference cycles are present.

	return diags
}

// getValidExprReferences gets the references from a particular expression,
// while also performing basic static validation on them to ensure that
// they refer to objects that are actually defined in the configuration.
func (c *Config) getValidExprReferences(expr hcl.Expression) ([]*addrs.ProjectConfigReference, tfdiags.Diagnostics) {
	if expr == nil {
		return nil, nil
	}

	traversals := expr.Variables()
	if len(traversals) == 0 {
		return nil, nil
	}

	ret := make([]*addrs.ProjectConfigReference, 0, len(traversals))
	var diags tfdiags.Diagnostics
	for _, traversal := range traversals {
		ref, moreDiags := addrs.ParseProjectConfigRef(traversal)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}

		switch addr := ref.Subject.(type) {
		case addrs.LocalValue:
			if _, ok := c.Locals[addr.Name]; !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared local value",
					Detail:   fmt.Sprintf("There is no local value named %q in this project configuration.", addr.Name),
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
				continue
			}
		case addrs.ProjectContextValue:
			if _, ok := c.Context[addr.Name]; !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared context value",
					Detail:   fmt.Sprintf("There is no context value named %q in this project configuration.", addr.Name),
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
				continue
			}
		case addrs.ForEachAttr:
			if addr.Name != "key" && addr.Name != "value" {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid \"each\" object attribute",
					Detail:   "The valid \"each\" object attributes are either each.key or each.value.",
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
				continue
			}
		case addrs.ProjectWorkspace:
			if _, ok := c.Workspaces[addr.Name]; !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared workspace block",
					Detail:   fmt.Sprintf("There is no workspace %q block in this project configuration.", addr.Name),
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
				continue
			}
		case addrs.ProjectUpstreamWorkspace:
			if _, ok := c.Upstreams[addr.Name]; !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undeclared upstream workspace block",
					Detail:   fmt.Sprintf("There is no upstream %q block in this project configuration.", addr.Name),
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
				continue
			}
		}

		ret = append(ret, ref)
	}

	return ret, diags
}
