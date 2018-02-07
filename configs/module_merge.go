package configs

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

// The methods in this file are used by Module.mergeFile to apply overrides
// to our different configuration elements. These methods all follow the
// pattern of mutating the receiver to incorporate settings from the parameter,
// returning error diagnostics if any aspect of the parameter cannot be merged
// into the receiver for some reason.
//
// User expectation is that anything _explicitly_ set in the given object
// should take precedence over the corresponding settings in the receiver,
// but that anything omitted in the given object should be left unchanged.
// In some cases it may be reasonable to do a "deep merge" of certain nested
// features, if it is possible to unambiguously correlate the nested elements
// and their behaviors are orthogonal to each other.

func (p *Provider) merge(op *Provider) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if op.Version.Required != nil {
		p.Version = op.Version
	}

	p.Config = mergeBodies(p.Config, op.Config)

	return diags
}

func mergeProviderVersionConstraints(recv map[string][]VersionConstraint, ovrd []*ProviderRequirement) {
	// Any provider name that's mentioned in the override gets nilled out in
	// our map so that we'll rebuild it below. Any provider not mentioned is
	// left unchanged.
	for _, reqd := range ovrd {
		delete(recv, reqd.Name)
	}
	for _, reqd := range ovrd {
		recv[reqd.Name] = append(recv[reqd.Name], reqd.Requirement)
	}
}

func (v *Variable) merge(ov *Variable) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if ov.Description != "" {
		v.Description = ov.Description
	}
	if ov.Default != cty.NilVal {
		v.Default = ov.Default
	}
	if ov.TypeHint != TypeHintNone {
		v.TypeHint = ov.TypeHint
	}

	return diags
}

func (l *Local) merge(ol *Local) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// Since a local is just a single expression in configuration, the
	// override definition entirely replaces the base definition, including
	// the source range so that we'll send the user to the right place if
	// there is an error.
	l.Expr = ol.Expr
	l.DeclRange = ol.DeclRange

	return diags
}

func (o *Output) merge(oo *Output) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if oo.Description != "" {
		o.Description = oo.Description
	}
	if oo.Expr != nil {
		o.Expr = oo.Expr
	}
	if oo.Sensitive {
		// Since this is just a bool, we can't distinguish false from unset
		// and so the override can only make the output _more_ sensitive.
		o.Sensitive = oo.Sensitive
	}

	// We don't allow depends_on to be overridden because that is likely to
	// cause confusing misbehavior.
	if len(oo.DependsOn) != 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported override",
			Detail:   "The depends_on argument may not be overridden.",
			Subject:  oo.DependsOn[0].SourceRange().Ptr(), // the first item is the closest range we have
		})
	}

	return diags
}

func (mc *ModuleCall) merge(omc *ModuleCall) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if omc.SourceAddr != "" {
		mc.SourceAddr = omc.SourceAddr
		mc.SourceAddrRange = omc.SourceAddrRange
	}

	if omc.Count != nil {
		mc.Count = omc.Count
	}

	if omc.ForEach != nil {
		mc.ForEach = omc.ForEach
	}

	if len(omc.Version.Required) != 0 {
		mc.Version = omc.Version
	}

	mc.Config = mergeBodies(mc.Config, omc.Config)

	// We don't allow depends_on to be overridden because that is likely to
	// cause confusing misbehavior.
	if len(mc.DependsOn) != 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported override",
			Detail:   "The depends_on argument may not be overridden.",
			Subject:  mc.DependsOn[0].SourceRange().Ptr(), // the first item is the closest range we have
		})
	}

	return diags
}

func (r *ManagedResource) merge(or *ManagedResource) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if or.Connection != nil {
		r.Connection = or.Connection
	}
	if or.Count != nil {
		r.Count = or.Count
	}
	if or.CreateBeforeDestroy {
		// We can't distinguish false from unset here
		r.CreateBeforeDestroy = or.CreateBeforeDestroy
	}
	if or.ForEach != nil {
		r.ForEach = or.ForEach
	}
	if len(or.IgnoreChanges) != 0 {
		r.IgnoreChanges = or.IgnoreChanges
	}
	if or.PreventDestroy {
		// We can't distinguish false from unset here
		r.PreventDestroy = or.PreventDestroy
	}
	if or.ProviderConfigRef != nil {
		r.ProviderConfigRef = or.ProviderConfigRef
	}
	if len(or.Provisioners) != 0 {
		r.Provisioners = or.Provisioners
	}

	r.Config = mergeBodies(r.Config, or.Config)

	// We don't allow depends_on to be overridden because that is likely to
	// cause confusing misbehavior.
	if len(r.DependsOn) != 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported override",
			Detail:   "The depends_on argument may not be overridden.",
			Subject:  r.DependsOn[0].SourceRange().Ptr(), // the first item is the closest range we have
		})
	}

	return diags
}

func (r *DataResource) merge(or *DataResource) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if or.Count != nil {
		r.Count = or.Count
	}
	if or.ForEach != nil {
		r.ForEach = or.ForEach
	}
	if or.ProviderConfigRef != nil {
		r.ProviderConfigRef = or.ProviderConfigRef
	}

	r.Config = mergeBodies(r.Config, or.Config)

	// We don't allow depends_on to be overridden because that is likely to
	// cause confusing misbehavior.
	if len(r.DependsOn) != 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported override",
			Detail:   "The depends_on argument may not be overridden.",
			Subject:  r.DependsOn[0].SourceRange().Ptr(), // the first item is the closest range we have
		})
	}

	return diags
}
