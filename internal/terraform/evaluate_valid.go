// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
func (e *Evaluator) StaticValidateReferences(refs []*addrs.Reference, modAddr addrs.Module, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, ref := range refs {
		moreDiags := e.StaticValidateReference(ref, modAddr, self, source)
		diags = diags.Append(moreDiags)
	}
	return diags
}

func (e *Evaluator) StaticValidateReference(ref *addrs.Reference, modAddr addrs.Module, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
	modCfg := e.Config.Descendant(modAddr)
	if modCfg == nil {
		// This is a bug in the caller rather than a problem with the
		// reference, but rather than crashing out here in an unhelpful way
		// we'll just ignore it and trust a different layer to catch it.
		return nil
	}
	return e.staticValidateReference(ref, modCfg, self, source)
}

func (e *Evaluator) staticValidateReference(ref *addrs.Reference, modCfg *configs.Config, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
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
				Detail:  `The "self" object is not available in this context. This object can be used only in resource provisioner, connection, and postcondition blocks.`,
				Subject: ref.SourceRange.ToHCL().Ptr(),
			})
			return diags
		}

		synthRef := *ref // shallow copy
		synthRef.Subject = self
		ref = &synthRef
	}

	switch addr := ref.Subject.(type) {

	// For static validation we validate both resource and resource instance references the same way.
	// We mostly disregard the index, though we do some simple validation of
	// its _presence_ in staticValidateSingleResourceReference and
	// staticValidateMultiResourceReference respectively.
	case addrs.Resource:
		var diags tfdiags.Diagnostics
		diags = diags.Append(staticValidateSingleResourceReference(modCfg, addr, ref.Remaining, ref.SourceRange))
		diags = diags.Append(staticValidateResourceReference(modCfg, addr, source, e.Plugins, ref.Remaining, ref.SourceRange))
		return diags
	case addrs.ResourceInstance:
		var diags tfdiags.Diagnostics
		diags = diags.Append(staticValidateMultiResourceReference(modCfg, addr, ref.Remaining, ref.SourceRange))
		diags = diags.Append(staticValidateResourceReference(modCfg, addr.ContainingResource(), source, e.Plugins, ref.Remaining, ref.SourceRange))
		return diags

	// We also handle all module call references the same way, disregarding index.
	case addrs.ModuleCall:
		return staticValidateModuleCallReference(modCfg, addr, ref.Remaining, ref.SourceRange)
	case addrs.ModuleCallInstance:
		return staticValidateModuleCallReference(modCfg, addr.Call, ref.Remaining, ref.SourceRange)
	case addrs.ModuleCallInstanceOutput:
		// This one is a funny one because we will take the output name referenced
		// and use it to fake up a "remaining" that would make sense for the
		// module call itself, rather than for the specific output, and then
		// we can just re-use our static module call validation logic.
		remain := make(hcl.Traversal, len(ref.Remaining)+1)
		copy(remain[1:], ref.Remaining)
		remain[0] = hcl.TraverseAttr{
			Name: addr.Name,

			// Using the whole reference as the source range here doesn't exactly
			// match how HCL would normally generate an attribute traversal,
			// but is close enough for our purposes.
			SrcRange: ref.SourceRange.ToHCL(),
		}
		return staticValidateModuleCallReference(modCfg, addr.Call.Call, remain, ref.SourceRange)

	default:
		// Anything else we'll just permit through without any static validation
		// and let it be caught during dynamic evaluation, in evaluate.go .
		return nil
	}
}

func staticValidateSingleResourceReference(modCfg *configs.Config, addr addrs.Resource, remain hcl.Traversal, rng tfdiags.SourceRange) tfdiags.Diagnostics {
	// If we have at least one step in "remain" and this resource has
	// "count" set then we know for sure this in invalid because we have
	// something like:
	//     aws_instance.foo.bar
	// ...when we really need
	//     aws_instance.foo[count.index].bar

	// It is _not_ safe to do this check when remain is empty, because that
	// would also match aws_instance.foo[count.index].bar due to `count.index`
	// not being statically-resolvable as part of a reference, and match
	// direct references to the whole aws_instance.foo tuple.
	if len(remain) == 0 {
		return nil
	}

	var diags tfdiags.Diagnostics

	cfg := modCfg.Module.ResourceByAddr(addr)
	if cfg == nil {
		// We'll just bail out here and catch this in our subsequent call to
		// staticValidateResourceReference, then.
		return diags
	}

	if cfg.Count != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Missing resource instance key`,
			Detail:   fmt.Sprintf("Because %s has \"count\" set, its attributes must be accessed on specific instances.\n\nFor example, to correlate with indices of a referring resource, use:\n    %s[count.index]", addr, addr),
			Subject:  rng.ToHCL().Ptr(),
		})
	}
	if cfg.ForEach != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Missing resource instance key`,
			Detail:   fmt.Sprintf("Because %s has \"for_each\" set, its attributes must be accessed on specific instances.\n\nFor example, to correlate with indices of a referring resource, use:\n    %s[each.key]", addr, addr),
			Subject:  rng.ToHCL().Ptr(),
		})
	}

	return diags
}

func staticValidateMultiResourceReference(modCfg *configs.Config, addr addrs.ResourceInstance, remain hcl.Traversal, rng tfdiags.SourceRange) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	cfg := modCfg.Module.ResourceByAddr(addr.ContainingResource())
	if cfg == nil {
		// We'll just bail out here and catch this in our subsequent call to
		// staticValidateResourceReference, then.
		return diags
	}

	if addr.Key == addrs.NoKey {
		// This is a different path into staticValidateSingleResourceReference
		return staticValidateSingleResourceReference(modCfg, addr.ContainingResource(), remain, rng)
	} else {
		if cfg.Count == nil && cfg.ForEach == nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Unexpected resource instance key`,
				Detail:   fmt.Sprintf(`Because %s does not have "count" or "for_each" set, references to it must not include an index key. Remove the bracketed index to refer to the single instance of this resource.`, addr.ContainingResource()),
				Subject:  rng.ToHCL().Ptr(),
			})
		}
	}

	return diags
}

func staticValidateResourceReference(modCfg *configs.Config, addr addrs.Resource, source addrs.Referenceable, plugins *contextPlugins, remain hcl.Traversal, rng tfdiags.SourceRange) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	var modeAdjective string
	modeArticleUpper := "A"
	switch addr.Mode {
	case addrs.ManagedResourceMode:
		modeAdjective = "managed"
	case addrs.DataResourceMode:
		modeAdjective = "data"
	case addrs.EphemeralResourceMode:
		modeAdjective = "ephemeral"
		modeArticleUpper = "An"
	default:
		// should never happen
		modeAdjective = "<invalid-mode>"
	}

	cfg := modCfg.Module.ResourceByAddr(addr)
	if cfg == nil {
		var suggestion string
		// A common mistake is omitting the data. prefix when trying to refer
		// to a data resource, so we'll add a special hint for that.
		if addr.Mode == addrs.ManagedResourceMode {
			candidateAddr := addr // not a pointer, so this is a copy
			candidateAddr.Mode = addrs.DataResourceMode
			if candidateCfg := modCfg.Module.ResourceByAddr(candidateAddr); candidateCfg != nil {
				suggestion = fmt.Sprintf("\n\nDid you mean the data resource %s?", candidateAddr)
			}
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared resource`,
			Detail: fmt.Sprintf(
				`%s %s resource %q %q has not been declared in %s.%s`,
				modeArticleUpper, modeAdjective,
				addr.Type, addr.Name,
				moduleConfigDisplayAddr(modCfg.Path),
				suggestion,
			),
			Subject: rng.ToHCL().Ptr(),
		})
		return diags
	}

	if cfg.Container != nil && (source == nil || !cfg.Container.Accessible(source)) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to scoped resource`,
			Detail:   fmt.Sprintf(`The referenced %s resource %q %q is not available from this context.`, modeAdjective, addr.Type, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
	}

	providerFqn := modCfg.Module.ProviderForLocalConfig(cfg.ProviderConfigAddr())
	schema, _, err := plugins.ResourceTypeSchema(providerFqn, addr.Mode, addr.Type)
	if err != nil {
		// Prior validation should've taken care of a schema lookup error,
		// so we should never get here but we'll handle it here anyway for
		// robustness.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Failed provider schema lookup`,
			Detail:   fmt.Sprintf(`Couldn't load schema for %s resource type %q in %s: %s.`, modeAdjective, addr.Type, providerFqn.String(), err),
			Subject:  rng.ToHCL().Ptr(),
		})
	}

	if schema == nil {
		// Prior validation should've taken care of a resource block with an
		// unsupported type, so we should never get here but we'll handle it
		// here anyway for robustness.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid resource type`,
			Detail: fmt.Sprintf(
				`%s %s resource type %q is not supported by provider %q.`,
				modeArticleUpper, modeAdjective,
				addr.Type,
				providerFqn.String(),
			),
			Subject: rng.ToHCL().Ptr(),
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

func staticValidateModuleCallReference(modCfg *configs.Config, addr addrs.ModuleCall, remain hcl.Traversal, rng tfdiags.SourceRange) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// For now, our focus here is just in testing that the referenced module
	// call exists. All other validation is deferred until evaluation time.
	_, exists := modCfg.Module.ModuleCalls[addr.Name]
	if !exists {
		var suggestions []string
		for name := range modCfg.Module.ModuleCalls {
			suggestions = append(suggestions, name)
		}
		sort.Strings(suggestions)
		suggestion := didyoumean.NameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared module`,
			Detail:   fmt.Sprintf(`No module call named %q is declared in %s.%s`, addr.Name, moduleConfigDisplayAddr(modCfg.Path), suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return diags
	}

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
