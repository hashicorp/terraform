package moduletest

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Run struct {
	Config *configs.TestRun

	Name   string
	Status Status

	Diagnostics tfdiags.Diagnostics
}

func (run *Run) GetTargets() ([]addrs.Targetable, tfdiags.Diagnostics) {
	var diagnostics tfdiags.Diagnostics
	var targets []addrs.Targetable

	for _, target := range run.Config.Options.Target {
		addr, diags := addrs.ParseTarget(target)
		diagnostics = diagnostics.Append(diags)
		if addr != nil {
			targets = append(targets, addr.Subject)
		}
	}

	return targets, diagnostics
}

func (run *Run) GetReplaces() ([]addrs.AbsResourceInstance, tfdiags.Diagnostics) {
	var diagnostics tfdiags.Diagnostics
	var replaces []addrs.AbsResourceInstance

	for _, replace := range run.Config.Options.Replace {
		addr, diags := addrs.ParseAbsResourceInstance(replace)
		diagnostics = diagnostics.Append(diags)
		if diags.HasErrors() {
			continue
		}

		if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
			diagnostics = diagnostics.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Can only target managed resources for forced replacements.",
				Detail:   addr.String(),
				Subject:  replace.SourceRange().Ptr(),
			})
			continue
		}

		replaces = append(replaces, addr)
	}

	return replaces, diagnostics
}

func (run *Run) GetReferences() ([]*addrs.Reference, tfdiags.Diagnostics) {
	var diagnostics tfdiags.Diagnostics
	var references []*addrs.Reference

	for _, rule := range run.Config.CheckRules {
		for _, variable := range rule.Condition.Variables() {
			reference, diags := addrs.ParseRef(variable)
			diagnostics = diagnostics.Append(diags)
			if reference != nil {
				references = append(references, reference)
			}
		}
		for _, variable := range rule.ErrorMessage.Variables() {
			reference, diags := addrs.ParseRef(variable)
			diagnostics = diagnostics.Append(diags)
			if reference != nil {
				references = append(references, reference)
			}
		}
	}

	return references, diagnostics
}
