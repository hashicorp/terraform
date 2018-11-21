package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/tfdiags"
)

// StaticValidateReferences checks the given references against schemas and
// other statically-checkable rules, producing error diagnostics if any
// problems are found.
//
// If this method returns errors for a particular reference then evaluating
// that reference is likely to generate a very similar error, so callers should
// not run this method and then also evaluate the source expression(s) and
// merge the two sets of diagnostics together, since this will result in
// confusing redundant errors.
//
// This method can find more errors than can be found by evaluating an
// expression with a partially-populated scope, since it checks the referenced
// names directly against the schema rather than relying on evaluation errors.
//
// The result may include warning diagnostics if, for example, deprecated
// features are referenced.
func (d *evaluationStateData) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, ref := range refs {
		moreDiags := d.staticValidateReference(ref, self)
		diags = diags.Append(moreDiags)
	}
	return diags
}

func (d *evaluationStateData) staticValidateReference(ref *addrs.Reference, self addrs.Referenceable) tfdiags.Diagnostics {
	modCfg := d.Evaluator.Config.DescendentForInstance(d.ModulePath)
	if modCfg == nil {
		// This is a bug in the caller rather than a problem with the
		// reference, but rather than crashing out here in an unhelpful way
		// we'll just ignore it and trust a different layer to catch it.
		return nil
	}

	if ref.Subject == addrs.Self {
		// The "self" address is a special alias for the address given as
		// our self parameter here, if present.
		if self == nil {
			var diags tfdiags.Diagnostics
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid "self" reference`,
				// This detail message mentions some current practice that
				// this codepath doesn't really "know about". If the "self"
				// object starts being supported in more contexts later then
				// we'll need to adjust this message.
				Detail:  `The "self" object is not available in this context. This object can be used only in resource provisioner and connection blocks.`,
				Subject: ref.SourceRange.ToHCL().Ptr(),
			})
			return diags
		}

		synthRef := *ref // shallow copy
		synthRef.Subject = self
		ref = &synthRef
	}

	switch addr := ref.Subject.(type) {

	// For static validation we validate both resource and resource instance references the same way, disregarding the index
	case addrs.Resource:
		return d.staticValidateResourceReference(modCfg, addr, ref.Remaining, ref.SourceRange)
	case addrs.ResourceInstance:
		return d.staticValidateResourceReference(modCfg, addr.ContainingResource(), ref.Remaining, ref.SourceRange)

	default:
		// Anything else we'll just permit through without any static validation
		// and let it be caught during dynamic evaluation, in evaluate.go .
		return nil
	}
}

func (d *evaluationStateData) staticValidateResourceReference(modCfg *configs.Config, addr addrs.Resource, remain hcl.Traversal, rng tfdiags.SourceRange) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	var modeAdjective string
	switch addr.Mode {
	case addrs.ManagedResourceMode:
		modeAdjective = "managed"
	case addrs.DataResourceMode:
		modeAdjective = "data"
	default:
		// should never happen
		modeAdjective = "<invalid-mode>"
	}

	cfg := modCfg.Module.ResourceByAddr(addr)
	if cfg == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared resource`,
			Detail:   fmt.Sprintf(`A %s resource %q %q has not been declared in %s`, modeAdjective, addr.Type, addr.Name, moduleConfigDisplayAddr(modCfg.Path)),
			Subject:  rng.ToHCL().Ptr(),
		})
		return diags
	}

	// Normally accessing this directly is wrong because it doesn't take into
	// account provider inheritance, etc but it's okay here because we're only
	// paying attention to the type anyway.
	providerType := cfg.ProviderConfigAddr().Type
	var schema *configschema.Block
	switch addr.Mode {
	case addrs.ManagedResourceMode:
		schema = d.Evaluator.Schemas.ResourceTypeConfig(providerType, addr.Type)
	case addrs.DataResourceMode:
		schema = d.Evaluator.Schemas.DataSourceConfig(providerType, addr.Type)
	}

	if schema == nil {
		// Prior validation should've taken care of a resource block with an
		// unsupported type, so we should never get here but we'll handle it
		// here anyway for robustness.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid resource type`,
			Detail:   fmt.Sprintf(`A %s resource type %q is not supported by provider %q.`, modeAdjective, addr.Type, providerType),
			Subject:  rng.ToHCL().Ptr(),
		})
		return diags
	}

	// As a special case we'll detect attempts to access an attribute called
	// "count" and produce a special error for it, since versions of Terraform
	// prior to v0.12 offered this as a weird special case that we can no
	// longer support.
	if len(remain) > 0 {
		if step, ok := remain[0].(hcl.TraverseAttr); ok && step.Name == "count" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid resource count attribute`,
				Detail:   fmt.Sprintf(`The special "count" attribute is no longer supported after Terraform v0.12. Instead, use length(%s) to count resource instances.`, addr),
				Subject:  rng.ToHCL().Ptr(),
			})
			return diags
		}
	}

	// If we got this far then we'll try to validate the remaining traversal
	// steps against our schema.
	moreDiags := schema.StaticValidateTraversal(remain)
	diags = diags.Append(moreDiags)

	return diags
}

// moduleConfigDisplayAddr returns a string describing the given module
// address that is appropriate for returning to users in situations where the
// root module is possible. Specifically, it returns "the root module" if the
// root module instance is given, or a string representation of the module
// address otherwise.
func moduleConfigDisplayAddr(addr addrs.Module) string {
	switch {
	case addr.IsRoot():
		return "the root module"
	default:
		return addr.String()
	}
}
