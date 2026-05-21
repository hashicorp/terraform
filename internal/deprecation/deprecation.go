// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package deprecation

import (
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
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

// ValidateAndUnmark checks the given value for deprecation marks and returns the value
// with deprecation marks removed along with diagnostics for each deprecation found,
// unless deprecation warnings are suppressed for the given module.
//
// This is only appropriate for non-terminal values (values that can be referenced) and primitive
// values.
// If the value can not be referenced, use ValidateExpressionDeepAndUnmark or ValidateConfigAndUnmark instead.
func (d *Deprecations) ValidateAndUnmark(value cty.Value, module addrs.Module, rng *hcl.Range) (cty.Value, tfdiags.Diagnostics) {
	notDeprecatedValue, deprecationMarks := marks.GetDeprecationMarks(value)
	return notDeprecatedValue, d.deprecationMarksToDiagnostics(deprecationMarks, module, rng)
}

// ValidateExpressionDeepAndUnmark looks for deprecation marks deeply within the given value
// and returns the value with deprecation marks removed along with diagnostics for each
// deprecation found, unless deprecation warnings are suppressed for the given module.
// It finds the most specific range possible for each diagnostic.
func (d *Deprecations) ValidateExpressionDeepAndUnmark(value cty.Value, module addrs.Module, expr hcl.Expression) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	undeprecatedVal, pdms := marks.GetDeprecationMarksDeep(value)

	// Check if we need to suppress deprecation warnings for this module call.
	if d.IsModuleCallDeprecationSuppressed(module) {
		return undeprecatedVal, diags
	}

	for _, pdm := range pdms {
		rng := tfdiags.RangeForExpressionAtPath(expr, pdm.Path)
		diags = diags.Append(deprecationMarkToDiagnostic(pdm.Mark, &rng))
	}

	return undeprecatedVal, diags
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
		diags = diags.Append(deprecationMarkToDiagnostic(depMark, rng))
	}
	return diags
}

func deprecationMarkToDiagnostic(depMark marks.DeprecationMark, subject *hcl.Range) *hcl.Diagnostic {
	diag := &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Deprecated value used",
		Detail:   depMark.Message,
		Subject:  subject,
	}
	if depMark.OriginDescription != "" {
		diag.Extra = &tfdiags.DeprecationOriginDiagnosticExtra{
			OriginDescription: depMark.OriginDescription,
		}
	}
	return diag
}

// ValidateAndUnmarkConfig checks the given value deeply for deprecation marks and returns
// the value with deprecation marks removed along with diagnostics for each deprecation found,
// unless deprecation warnings are suppressed for the given module.
func (d *Deprecations) ValidateAndUnmarkConfig(value cty.Value, schema *configschema.Block, module addrs.Module) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	undeprecatedVal, pdms := marks.GetDeprecationMarksDeep(value)

	if d.IsModuleCallDeprecationSuppressed(module) {
		// Even if we don't want to get deprecation warnings we want to remove the marks
		return undeprecatedVal, diags
	}

	for _, pdm := range pdms {
		diag := tfdiags.AttributeValue(
			tfdiags.Warning,
			"Deprecated value used",
			pdm.Mark.Message,
			pdm.Path,
		)
		if pdm.Mark.OriginDescription != "" {
			diag = tfdiags.Override(
				diag,
				tfdiags.Warning, // We just want to override the extra info
				func() tfdiags.DiagnosticExtraWrapper {
					return &tfdiags.DeprecationOriginDiagnosticExtra{
						OriginDescription: pdm.Mark.OriginDescription,
					}
				})
		}
		diags = diags.Append(diag)
	}

	return undeprecatedVal, diags
}

func (d *Deprecations) IsModuleCallDeprecationSuppressed(addr addrs.Module) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, mod := range d.suppressedModules {
		if mod.TargetContains(addr) {
			return true
		}
	}
	return false
}
