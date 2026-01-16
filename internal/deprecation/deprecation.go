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

// Validate checks the given value for deprecation marks and returns diagnostics
// for each deprecation found, unless deprecation warnings are suppressed for the given module.

// This is only appropriate for non-terminal values (values that can be referenced) and primitive
// values.
// If the value can not be referenced, use ValidateDeep or ValidateAsConfig instead.
func (d *Deprecations) Validate(value cty.Value, module addrs.Module, rng *hcl.Range) (cty.Value, tfdiags.Diagnostics) {
	deprecationMarks := marks.GetDeprecationMarks(value)
	notDeprecatedValue := marks.RemoveDeprecationMarks(value)
	return notDeprecatedValue, d.deprecationMarksToDiagnostics(deprecationMarks, module, rng)
}

// ValidateDeep does the same as Validate but checks deeply nested deprecation marks as well.
func (d *Deprecations) ValidateDeep(value cty.Value, module addrs.Module, rng *hcl.Range) (cty.Value, tfdiags.Diagnostics) {
	deprecationMarks := marks.GetDeprecationMarksDeep(value)
	notDeprecatedValue := marks.RemoveDeprecationMarksDeep(value)
	return notDeprecatedValue, d.deprecationMarksToDiagnostics(deprecationMarks, module, rng)
}

func (d *Deprecations) deprecationMarksToDiagnostics(deprecationMarks []marks.DeprecationMark, module addrs.Module, rng *hcl.Range) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if len(deprecationMarks) == 0 {
		return diags
	}

	// Check if we need to suppress deprecation warnings for this module call.
	if d.IsModuleCallDeprecationSuppressed(module) {
		return diags
	}

	for _, depMark := range deprecationMarks {
		diag := &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Deprecated value used",
			Detail:   depMark.Message,
			Subject:  rng,
		}
		if depMark.OriginDescription != "" {
			diag.Extra = &tfdiags.DeprecationOriginDiagnosticExtra{
				OriginDescription: depMark.OriginDescription,
			}
		}
		diags = diags.Append(diag)
	}
	return diags
}

// ValidateAsConfig checks the given value for deprecation marks and returns diagnostics
// for each deprecation found, unless deprecation warnings are suppressed for the given module.
// It checks for deeply nested deprecation marks as well.
func (d *Deprecations) ValidateAsConfig(value cty.Value, module addrs.Module) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, pvms := value.UnmarkDeepWithPaths()

	if len(pvms) == 0 || d.IsModuleCallDeprecationSuppressed(module) {
		return diags
	}

	for _, pvm := range pvms {
		for m := range pvm.Marks {
			if depMark, ok := m.(marks.DeprecationMark); ok {
				diag := tfdiags.AttributeValue(
					tfdiags.Warning,
					"Deprecated value used",
					depMark.Message,
					pvm.Path,
				)
				if depMark.OriginDescription != "" {
					diag = tfdiags.Override(
						diag,
						tfdiags.Warning, // We just want to override the extra info
						func() tfdiags.DiagnosticExtraWrapper {
							return &tfdiags.DeprecationOriginDiagnosticExtra{
								// TODO: Remove common prefixes from origin descriptions?
								OriginDescription: depMark.OriginDescription,
							}
						})
				}

				diags = diags.Append(diag)

			}
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
