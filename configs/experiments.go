package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/experiments"
)

// sniffActiveExperiments does minimal parsing of the given body for
// "terraform" blocks with "experiments" attributes, returning the
// experiments found.
//
// This is separate from other processing so that we can be sure that all of
// the experiments are known before we process the result of the module config,
// and thus we can take into account which experiments are active when deciding
// how to decode.
func sniffActiveExperiments(body hcl.Body) (experiments.Set, hcl.Diagnostics) {
	rootContent, _, diags := body.PartialContent(configFileTerraformBlockSniffRootSchema)

	ret := experiments.NewSet()

	for _, block := range rootContent.Blocks {
		content, _, blockDiags := block.Body.PartialContent(configFileExperimentsSniffBlockSchema)
		diags = append(diags, blockDiags...)

		attr, exists := content.Attributes["experiments"]
		if !exists {
			continue
		}

		exps, expDiags := decodeExperimentsAttr(attr)
		diags = append(diags, expDiags...)
		if !expDiags.HasErrors() {
			ret = experiments.SetUnion(ret, exps)
		}
	}

	return ret, diags
}

func decodeExperimentsAttr(attr *hcl.Attribute) (experiments.Set, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	exprs, moreDiags := hcl.ExprList(attr.Expr)
	diags = append(diags, moreDiags...)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	var ret = experiments.NewSet()
	for _, expr := range exprs {
		kw := hcl.ExprAsKeyword(expr)
		if kw == "" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid experiment keyword",
				Detail:   "Elements of \"experiments\" must all be keywords representing active experiments.",
				Subject:  expr.Range().Ptr(),
			})
			continue
		}

		exp, err := experiments.GetCurrent(kw)
		switch err := err.(type) {
		case experiments.UnavailableError:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unknown experiment keyword",
				Detail:   fmt.Sprintf("There is no current experiment with the keyword %q.", kw),
				Subject:  expr.Range().Ptr(),
			})
		case experiments.ConcludedError:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Experiment has concluded",
				Detail:   fmt.Sprintf("Experiment %q is no longer available. %s", kw, err.Message),
				Subject:  expr.Range().Ptr(),
			})
		case nil:
			// No error at all means it's valid and current.
			ret.Add(exp)

			// However, experimental features are subject to breaking changes
			// in future releases, so we'll warn about them to help make sure
			// folks aren't inadvertently using them in places where that'd be
			// inappropriate, particularly if the experiment is active in a
			// shared module they depend on.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  fmt.Sprintf("Experimental feature %q is active", exp.Keyword()),
				Detail:   "Experimental features are subject to breaking changes in future minor or patch releases, based on feedback.\n\nIf you have feedback on the design of this feature, please open a GitHub issue to discuss it.",
				Subject:  expr.Range().Ptr(),
			})

		default:
			// This should never happen, because GetCurrent is not documented
			// to return any other error type, but we'll handle it to be robust.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid experiment keyword",
				Detail:   fmt.Sprintf("Could not parse %q as an experiment keyword: %s.", kw, err.Error()),
				Subject:  expr.Range().Ptr(),
			})
		}
	}
	return ret, diags
}

func checkModuleExperiments(m *Module) hcl.Diagnostics {
	var diags hcl.Diagnostics

	// When we have current experiments, this is a good place to check that
	// the features in question can only be used when the experiments are
	// active. Return error diagnostics if a feature is being used without
	// opting in to the feature. For example:
	/*
		if !m.ActiveExperiments.Has(experiments.ResourceForEach) {
			for _, rc := range m.ManagedResources {
				if rc.ForEach != nil {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Resource for_each is experimental",
						Detail:   "This feature is currently an opt-in experiment, subject to change in future releases based on feedback.\n\nActivate the feature for this module by adding resource_for_each to the list of active experiments.",
						Subject:  rc.ForEach.Range().Ptr(),
					})
				}
			}
			for _, rc := range m.DataResources {
				if rc.ForEach != nil {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Resource for_each is experimental",
						Detail:   "This feature is currently an opt-in experiment, subject to change in future releases based on feedback.\n\nActivate the feature for this module by adding resource_for_each to the list of active experiments.",
						Subject:  rc.ForEach.Range().Ptr(),
					})
				}
			}
		}
	*/

	return diags
}
