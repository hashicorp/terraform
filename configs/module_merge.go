package configs

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
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

	p.Config = MergeBodies(p.Config, op.Config)

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

	if ov.DescriptionSet {
		v.Description = ov.Description
		v.DescriptionSet = ov.DescriptionSet
	}
	if ov.Default != cty.NilVal {
		v.Default = ov.Default
	}
	if ov.Type != cty.NilType {
		v.Type = ov.Type
	}
	if ov.ParsingMode != 0 {
		v.ParsingMode = ov.ParsingMode
	}

	// If the override file overrode type without default or vice-versa then
	// it may have created an invalid situation, which we'll catch now by
	// attempting to re-convert the value.
	//
	// Note that here we may be re-converting an already-converted base value
	// from the base config. This will be a no-op if the type was not changed,
	// but in particular might be user-observable in the edge case where the
	// literal value in config could've been converted to the overridden type
	// constraint but the converted value cannot. In practice, this situation
	// should be rare since most of our conversions are interchangable.
	if v.Default != cty.NilVal {
		val, err := convert.Convert(v.Default, v.Type)
		if err != nil {
			// What exactly we'll say in the error message here depends on whether
			// it was Default or Type that was overridden here.
			switch {
			case ov.Type != cty.NilType && ov.Default == cty.NilVal:
				// If only the type was overridden
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid default value for variable",
					Detail:   fmt.Sprintf("Overriding this variable's type constraint has made its default value invalid: %s.", err),
					Subject:  &ov.DeclRange,
				})
			case ov.Type == cty.NilType && ov.Default != cty.NilVal:
				// Only the default was overridden
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid default value for variable",
					Detail:   fmt.Sprintf("The overridden default value for this variable is not compatible with the variable's type constraint: %s.", err),
					Subject:  &ov.DeclRange,
				})
			default:
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid default value for variable",
					Detail:   fmt.Sprintf("This variable's default value is not compatible with its type constraint: %s.", err),
					Subject:  &ov.DeclRange,
				})
			}
		} else {
			v.Default = val
		}
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
	if oo.SensitiveSet {
		o.Sensitive = oo.Sensitive
		o.SensitiveSet = oo.SensitiveSet
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

	if omc.SourceSet {
		mc.SourceAddr = omc.SourceAddr
		mc.SourceAddrRange = omc.SourceAddrRange
		mc.SourceSet = omc.SourceSet
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

	mc.Config = MergeBodies(mc.Config, omc.Config)

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

func (r *Resource) merge(or *Resource) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if r.Mode != or.Mode {
		// This is always a programming error, since managed and data resources
		// are kept in separate maps in the configuration structures.
		panic(fmt.Errorf("can't merge %s into %s", or.Mode, r.Mode))
	}

	if or.Count != nil {
		r.Count = or.Count
	}
	if or.ForEach != nil {
		r.ForEach = or.ForEach
	}
	if or.ProviderConfigRef != nil {
		r.ProviderConfigRef = or.ProviderConfigRef
	}
	if r.Mode == addrs.ManagedResourceMode {
		// or.Managed is always non-nil for managed resource mode

		if or.Managed.Connection != nil {
			r.Managed.Connection = or.Managed.Connection
		}
		if or.Managed.CreateBeforeDestroySet {
			r.Managed.CreateBeforeDestroy = or.Managed.CreateBeforeDestroy
			r.Managed.CreateBeforeDestroySet = or.Managed.CreateBeforeDestroySet
		}
		if len(or.Managed.IgnoreChanges) != 0 {
			r.Managed.IgnoreChanges = or.Managed.IgnoreChanges
		}
		if or.Managed.PreventDestroySet {
			r.Managed.PreventDestroy = or.Managed.PreventDestroy
			r.Managed.PreventDestroySet = or.Managed.PreventDestroySet
		}
		if len(or.Managed.Provisioners) != 0 {
			r.Managed.Provisioners = or.Managed.Provisioners
		}
	}

	r.Config = MergeBodies(r.Config, or.Config)

	// We don't allow depends_on to be overridden because that is likely to
	// cause confusing misbehavior.
	if len(or.DependsOn) != 0 {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported override",
			Detail:   "The depends_on argument may not be overridden.",
			Subject:  or.DependsOn[0].SourceRange().Ptr(), // the first item is the closest range we have
		})
	}

	return diags
}
