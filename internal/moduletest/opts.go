// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func GetRunTargets(config *configs.TestRun) ([]addrs.Targetable, tfdiags.Diagnostics) {
	var diagnostics tfdiags.Diagnostics
	var targets []addrs.Targetable

	for _, target := range config.Options.Target {
		addr, diags := addrs.ParseTarget(target)
		diagnostics = diagnostics.Append(diags)
		if addr != nil {
			targets = append(targets, addr.Subject)
		}
	}

	return targets, diagnostics
}

func GetRunReplaces(config *configs.TestRun) ([]addrs.AbsResourceInstance, tfdiags.Diagnostics) {
	var diagnostics tfdiags.Diagnostics
	var replaces []addrs.AbsResourceInstance

	for _, replace := range config.Options.Replace {
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

func GetRunReferences(config *configs.TestRun) ([]*addrs.Reference, tfdiags.Diagnostics) {
	var diagnostics tfdiags.Diagnostics
	var references []*addrs.Reference

	for _, rule := range config.CheckRules {
		for _, variable := range rule.Condition.Variables() {
			reference, diags := addrs.ParseRefFromTestingScope(variable)
			diagnostics = diagnostics.Append(diags)
			if reference != nil {
				references = append(references, reference)
			}
		}
		for _, variable := range rule.ErrorMessage.Variables() {
			reference, diags := addrs.ParseRefFromTestingScope(variable)
			diagnostics = diagnostics.Append(diags)
			if reference != nil {
				references = append(references, reference)
			}
		}
	}

	for _, expr := range config.Variables {
		moreRefs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
		diagnostics = diagnostics.Append(moreDiags)
		references = append(references, moreRefs...)
	}

	return references, diagnostics
}
