// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deprecation

import (
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Deprecations keeps track of meta-information related to deprecation, e.g. which module calls
// suppress deprecation warnings.
type Deprecations struct {
	// Must hold this lock when accessing all fields after this one.
	mu sync.Mutex

	suppressedModules addrs.Set[addrs.Module]
}

func NewDeprecations() *Deprecations {
	return &Deprecations{
		suppressedModules: addrs.MakeSet[addrs.Module](),
	}
}

func (d *Deprecations) SuppressModuleCallDeprecation(addr addrs.Module) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.suppressedModules.Add(addr)
}

func (d *Deprecations) Validate(value cty.Value, module addrs.Module, rng *hcl.Range) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	deprecationMarks := marks.GetDeprecationMarks(value)
	if len(deprecationMarks) == 0 {
		return value, diags
	}

	notDeprecatedValue := marks.RemoveDeprecationMarks(value)
	diags = diags.Append(d.diagnosticsForDeprecationMarks(deprecationMarks, module, rng))
	return notDeprecatedValue, diags
}

func (d *Deprecations) ValidateAsConfig(value cty.Value, module addrs.Module) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, pvms := value.UnmarkDeepWithPaths()

	if len(pvms) == 0 || d.IsModuleCallDeprecationSuppressed(module) {
		return diags
	}

	for _, pvm := range pvms {
		for m := range pvm.Marks {
			if depMark, ok := m.(marks.DeprecationMark); ok {
				diags = diags.Append(
					tfdiags.AttributeValue(
						tfdiags.Warning,
						"Deprecated value used",
						depMark.Message,
						pvm.Path,
					),
				)
			}
		}
	}
	return diags
}

func (d *Deprecations) DiagnosticsForValueMarks(valueMarks cty.ValueMarks, module addrs.Module, rng *hcl.Range) tfdiags.Diagnostics {
	return d.diagnosticsForDeprecationMarks(marks.FilterDeprecationMarks(valueMarks), module, rng)
}

func (d *Deprecations) diagnosticsForDeprecationMarks(deprecationMarks []marks.DeprecationMark, module addrs.Module, rng *hcl.Range) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	// Check if we need to suppress deprecation warnings for this module call.
	if !d.IsModuleCallDeprecationSuppressed(module) {
		for _, depMark := range deprecationMarks {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Deprecated value used",
				Detail:   depMark.Message,
				Subject:  rng,
			})
		}
	}

	return diags
}

func (d *Deprecations) IsModuleCallDeprecationSuppressed(addr addrs.Module) bool {
	for _, mod := range d.suppressedModules {
		if mod.TargetContains(addr) {
			return true
		}
	}
	return false
}
