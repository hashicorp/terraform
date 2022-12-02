package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/experiments"
	"github.com/hashicorp/terraform/version"
)

// When developing UI for experimental features, you can temporarily disable
// the experiment warning by setting this package-level variable to a non-empty
// value using a link-time flag:
//
// go install -ldflags="-X 'github.com/hashicorp/terraform/internal/configs.disableExperimentWarnings=yes'"
//
// This functionality is for development purposes only and is not a feature we
// are committing to supporting for end users.
var disableExperimentWarnings = ""

// sniffActiveExperiments does minimal parsing of the given body for
// "terraform" blocks with "experiments" attributes, returning the
// experiments found.
//
// This is separate from other processing so that we can be sure that all of
// the experiments are known before we process the result of the module config,
// and thus we can take into account which experiments are active when deciding
// how to decode.
func sniffActiveExperiments(body hcl.Body, allowed bool) (experiments.Set, hcl.Diagnostics) {
	rootContent, _, diags := body.PartialContent(configFileTerraformBlockSniffRootSchema)

	ret := experiments.NewSet()

	for _, block := range rootContent.Blocks {
		content, _, blockDiags := block.Body.PartialContent(configFileExperimentsSniffBlockSchema)
		diags = append(diags, blockDiags...)

		if attr, exists := content.Attributes["language"]; exists {
			// We don't yet have a sense of selecting an edition of the
			// language, but we're reserving this syntax for now so that
			// if and when we do this later older versions of Terraform
			// will emit a more helpful error message than just saying
			// this attribute doesn't exist. Handling this as part of
			// experiments is a bit odd for now but justified by the
			// fact that a future fuller implementation of switchable
			// languages would be likely use a similar implementation
			// strategy as experiments, and thus would lead to this
			// function being refactored to deal with both concerns at
			// once. We'll see, though!
			kw := hcl.ExprAsKeyword(attr.Expr)
			currentVersion := version.SemVer.String()
			const firstEdition = "TF2021"
			switch {
			case kw == "": // (the expression wasn't a keyword at all)
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid language edition",
					Detail: fmt.Sprintf(
						"The language argument expects a bare language edition keyword. Terraform %s supports only language edition %s, which is the default.",
						currentVersion, firstEdition,
					),
					Subject: attr.Expr.Range().Ptr(),
				})
			case kw != firstEdition:
				rel := "different"
				if kw > firstEdition { // would be weird for this not to be true, but it's user input so anything goes
					rel = "newer"
				}
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported language edition",
					Detail: fmt.Sprintf(
						"Terraform v%s only supports language edition %s. This module requires a %s version of Terraform CLI.",
						currentVersion, firstEdition, rel,
					),
					Subject: attr.Expr.Range().Ptr(),
				})
			}
		}

		attr, exists := content.Attributes["experiments"]
		if !exists {
			continue
		}

		exps, expDiags := decodeExperimentsAttr(attr)

		// Because we concluded this particular experiment in the same
		// release as we made experiments alpha-releases-only, we need to
		// treat it as special to avoid masking the "experiment has concluded"
		// error with the more general "experiments are not available at all"
		// error. Note that this experiment is marked as concluded so this
		// only "allows" showing the different error message that it is
		// concluded, and does not allow actually using the experiment outside
		// of an alpha.
		// NOTE: We should be able to remove this special exception a release
		// or two after v1.3 when folks have had a chance to notice that the
		// experiment has concluded and update their modules accordingly.
		// When we do so, we might also consider changing decodeExperimentsAttr
		// to _not_ include concluded experiments in the returned set, since
		// we're doing that right now only to make this condition work.
		if exps.Has(experiments.ModuleVariableOptionalAttrs) && len(exps) == 1 {
			allowed = true
		}

		if allowed {
			diags = append(diags, expDiags...)
			if !expDiags.HasErrors() {
				ret = experiments.SetUnion(ret, exps)
			}
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Module uses experimental features",
				Detail:   "Experimental features are intended only for gathering early feedback on new language designs, and so are available only in alpha releases of Terraform.",
				Subject:  attr.NameRange.Ptr(),
			})
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
			// As a special case we still include the optional attributes
			// experiment if it's present, because our caller treats that
			// as special. See the comment in sniffActiveExperiments for
			// more information, and remove this special case here one the
			// special case up there is also removed.
			if kw == "module_variable_optional_attrs" {
				ret.Add(experiments.ModuleVariableOptionalAttrs)
			}

			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Experiment has concluded",
				Detail:   fmt.Sprintf("Experiment %q is no longer available. %s", kw, err.Message),
				Subject:  expr.Range().Ptr(),
			})
		case nil:
			// No error at all means it's valid and current.
			ret.Add(exp)

			if disableExperimentWarnings == "" {
				// However, experimental features are subject to breaking changes
				// in future releases, so we'll warn about them to help make sure
				// folks aren't inadvertently using them in places where that'd be
				// inappropriate, particularly if the experiment is active in a
				// shared module they depend on.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  fmt.Sprintf("Experimental feature %q is active", exp.Keyword()),
					Detail:   "Experimental features are available only in alpha releases of Terraform and are subject to breaking changes or total removal in later versions, based on feedback. We recommend against using experimental features in production.\n\nIf you have feedback on the design of this feature, please open a GitHub issue to discuss it.",
					Subject:  expr.Range().Ptr(),
				})
			}

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

	if !m.ActiveExperiments.Has(experiments.SmokeTests) {
		for _, st := range m.SmokeTests {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Smoke tests are experimental",
				Detail:   "This feature is currently an opt-in experiment, subject to change in future releases based on feedback.\n\nActivate the feature for this module by adding smoke_tests to the set of active experiments.",
				Subject:  st.DeclRange.Ptr(),
			})
		}
	}

	return diags
}
