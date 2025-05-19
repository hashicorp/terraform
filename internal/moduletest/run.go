// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

import (
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const (
	MainStateIdentifier = ""
)

type Run struct {
	Config *configs.TestRun

	// ModuleConfig is a copy of the module configuration that the run is testing.
	// The variables and provider configurations are copied so that the run can
	// modify them safely without affecting the original configuration.
	// However, any other fields in the module configuration are still shared between
	// all runs that use the same module configuration.
	ModuleConfig *configs.Config

	Verbose *Verbose

	Name   string
	Index  int
	Status Status

	Diagnostics tfdiags.Diagnostics

	// ExecutionMeta captures metadata about how the test run was executed.
	//
	// This field is not always populated. A run that has never been executed
	// will definitely have a nil value for this field. A run that was
	// executed may or may not populate this field, depending on exactly what
	// happened during the run execution. Callers accessing this field MUST
	// check for nil and handle that case in some reasonable way.
	//
	// Executing the same run multiple times may or may not update this field
	// on each execution.
	ExecutionMeta *RunExecutionMeta
}

func NewRun(config *configs.TestRun, moduleConfig *configs.Config, index int) *Run {
	// Make a copy the module configuration variables and provider configuration maps
	// so that the run can modify the map safely.
	newModuleConfig := *moduleConfig
	if moduleConfig.Module != nil {
		newModule := *moduleConfig.Module
		newModule.Variables = make(map[string]*configs.Variable, len(moduleConfig.Module.Variables))
		for name, variable := range moduleConfig.Module.Variables {
			newModule.Variables[name] = variable
		}
		newModule.ProviderConfigs = make(map[string]*configs.Provider, len(moduleConfig.Module.ProviderConfigs))
		for name, provider := range moduleConfig.Module.ProviderConfigs {
			newModule.ProviderConfigs[name] = provider
		}
		newModuleConfig.Module = &newModule
	}

	return &Run{
		Config:       config,
		ModuleConfig: &newModuleConfig,
		Name:         config.Name,
		Index:        index,
	}
}

type RunExecutionMeta struct {
	Start    time.Time
	Duration time.Duration
}

// StartTimestamp returns the start time metadata as a timestamp formatted as YYYY-MM-DDTHH:MM:SSZ.
// Times are converted to UTC, if they aren't already.
// If the start time is unset an empty string is returned.
func (m *RunExecutionMeta) StartTimestamp() string {
	if m.Start.IsZero() {
		return ""
	}
	return m.Start.UTC().Format(time.RFC3339)
}

// Verbose is a utility struct that holds all the information required for a run
// to render the results verbosely.
//
// At the moment, this basically means printing out the plan. To do that we need
// all the information within this struct.
type Verbose struct {
	Plan         *plans.Plan
	State        *states.State
	Config       *configs.Config
	Providers    map[addrs.Provider]providers.ProviderSchema
	Provisioners map[string]*configschema.Block
}

func (run *Run) Addr() addrs.Run {
	return addrs.Run{Name: run.Name}
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

	for _, expr := range run.Config.Variables {
		moreRefs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, expr)
		diagnostics = diagnostics.Append(moreDiags)
		references = append(references, moreRefs...)
	}

	return references, diagnostics
}

// GetStateKey returns the run's state key. If an explicit state key is set in
// the run's configuration, that key is returned. Otherwise, if the run is using
// an alternate module under test, the source of that module is returned as the
// state key. If neither of these conditions are met, an empty string is
// returned, and this denotes that the run is using the root module under test.
func (run *Run) GetStateKey() string {
	if run.Config.StateKey != "" {
		return run.Config.StateKey
	}

	// The run has an alternate module under test, so we can use the module's source
	if run.Config.ConfigUnderTest != nil {
		return run.Config.Module.Source.String()
	}

	return MainStateIdentifier
}

// GetModuleConfigID returns the identifier for the module configuration that
// this run is testing. This is used to uniquely identify the module
// configuration in the test state.
func (run *Run) GetModuleConfigID() string {
	return run.ModuleConfig.Module.SourceDir
}

// ExplainExpectedFailures is similar to ValidateExpectedFailures except it
// looks for any diagnostics produced by custom conditions and are included in
// the expected failures and adds an additional explanation that clarifies the
// expected failures are being ignored this time round.
//
// Generally, this function is used during an `apply` operation to explain that
// an expected failure during the planning stage will still result in the
// overall test failing as the plan failed and we couldn't even execute the
// apply stage.
func (run *Run) ExplainExpectedFailures(originals tfdiags.Diagnostics) tfdiags.Diagnostics {

	// We're going to capture all the checkable objects that are referenced
	// from the expected failures.
	expectedFailures := addrs.MakeMap[addrs.Referenceable, bool]()
	sourceRanges := addrs.MakeMap[addrs.Referenceable, tfdiags.SourceRange]()

	for _, traversal := range run.Config.ExpectFailures {
		// Ignore the diagnostics returned from the reference parsing, these
		// references will have been checked earlier in the process by the
		// validate stage so we don't need to do that again here.
		reference, _ := addrs.ParseRefFromTestingScope(traversal)
		expectedFailures.Put(reference.Subject, false)
		sourceRanges.Put(reference.Subject, reference.SourceRange)
	}

	var diags tfdiags.Diagnostics
	for _, diag := range originals {
		if diag.Severity() == tfdiags.Warning {
			// Then it's fine, the test will carry on without us doing anything.
			diags = diags.Append(diag)
			continue
		}

		if rule, ok := addrs.DiagnosticOriginatesFromCheckRule(diag); ok {

			var rng *hcl.Range
			expected := false
			switch rule.Container.CheckableKind() {
			case addrs.CheckableOutputValue:
				addr := rule.Container.(addrs.AbsOutputValue)
				if !addr.Module.IsRoot() {
					// failures can only be expected against checkable objects
					// in the root module. This diagnostic will be added into
					// returned set below.
					break
				}
				if expectedFailures.Has(addr.OutputValue) {
					expected = true
					rng = sourceRanges.Get(addr.OutputValue).ToHCL().Ptr()
				}

			case addrs.CheckableInputVariable:
				addr := rule.Container.(addrs.AbsInputVariableInstance)
				if !addr.Module.IsRoot() {
					// failures can only be expected against checkable objects
					// in the root module. This diagnostic will be added into
					// returned set below.
					break
				}
				if expectedFailures.Has(addr.Variable) {
					expected = true
				}

			case addrs.CheckableResource:
				addr := rule.Container.(addrs.AbsResourceInstance)
				if !addr.Module.IsRoot() {
					// failures can only be expected against checkable objects
					// in the root module. This diagnostic will be added into
					// returned set below.
					break
				}
				if expectedFailures.Has(addr.Resource) {
					expected = true
				}

				if expectedFailures.Has(addr.Resource.Resource) {
					expected = true
				}

			case addrs.CheckableCheck:
				// Check blocks only produce warnings so this branch shouldn't
				// ever be triggered anyway.
			default:
				panic("unrecognized CheckableKind: " + rule.Container.CheckableKind().String())
			}

			if expected {
				// Then this diagnostic was produced by a custom condition that
				// was expected to fail. But, it happened at the wrong time (eg.
				// we're trying to run an apply operation and this condition
				// failed during the plan so the overall test operation still
				// fails).
				//
				// We'll add a warning diagnostic explaining why the overall
				// test is still failing even though the error was expected, and
				// then add the original error into our diagnostics directly
				// after.

				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Expected failure while planning",
					Detail:   fmt.Sprintf("A custom condition within %s failed during the planning stage and prevented the requested apply operation. While this was an expected failure, the apply operation could not be executed and so the overall test case will be marked as a failure and the original diagnostic included in the test report.", rule.Container.String()),
					Subject:  rng,
				})
				diags = diags.Append(diag)
				continue
			}
		}

		// Otherwise, there is nothing special about this diagnostic so just
		// carry it through.
		diags = diags.Append(diag)
	}
	return diags
}

// ValidateExpectedFailures steps through the provided diagnostics (which should
// be the result of a plan or an apply operation), and does 3 things:
//  1. Removes diagnostics that match the expected failures from the config.
//  2. Upgrades warnings from check blocks into errors where appropriate so the
//     test will fail later.
//  3. Adds diagnostics for any expected failures that were not satisfied.
//
// Point 2 is a bit complicated so worth expanding on. In normal Terraform
// execution, any error that originates within a check block (either from an
// assertion or a scoped data source) is wrapped up as a Warning to be
// identified to the user but not to fail the actual Terraform operation. During
// test execution, we want to upgrade (or rollback) these warnings into errors
// again so the test will fail. We do that as part of this function as we are
// already processing the diagnostics from check blocks in here anyway.
//
// The way the function works out which diagnostics are relevant to expected
// failures is by using the tfdiags Extra functionality to detect which
// diagnostics were generated by custom conditions. Terraform adds the
// addrs.CheckRule that generated each diagnostic to the diagnostic itself so we
// can tell which diagnostics can be expected.
func (run *Run) ValidateExpectedFailures(originals tfdiags.Diagnostics) tfdiags.Diagnostics {

	// We're going to capture all the checkable objects that are referenced
	// from the expected failures.
	expectedFailures := addrs.MakeMap[addrs.Referenceable, bool]()
	sourceRanges := addrs.MakeMap[addrs.Referenceable, tfdiags.SourceRange]()

	for _, traversal := range run.Config.ExpectFailures {
		// Ignore the diagnostics returned from the reference parsing, these
		// references will have been checked earlier in the process by the
		// validate stage so we don't need to do that again here.
		reference, _ := addrs.ParseRefFromTestingScope(traversal)
		expectedFailures.Put(reference.Subject, false)
		sourceRanges.Put(reference.Subject, reference.SourceRange)
	}

	var diags tfdiags.Diagnostics
	for _, diag := range originals {

		if rule, ok := addrs.DiagnosticOriginatesFromCheckRule(diag); ok {
			switch rule.Container.CheckableKind() {
			case addrs.CheckableOutputValue:
				addr := rule.Container.(addrs.AbsOutputValue)
				if !addr.Module.IsRoot() {
					// failures can only be expected against checkable objects
					// in the root module. This diagnostic will be added into
					// returned set below.
					break
				}

				if diag.Severity() == tfdiags.Warning {
					// Warnings don't count as errors. This diagnostic will be
					// added into the returned set below.
					break
				}

				if expectedFailures.Has(addr.OutputValue) {
					// Then this failure is expected! Mark the original map as
					// having found a failure and swallow this error by
					// continuing and not adding it into the returned set of
					// diagnostics.
					expectedFailures.Put(addr.OutputValue, true)
					continue
				}

				// Otherwise, this isn't an expected failure so just fall out
				// and add it into the returned set of diagnostics below.

			case addrs.CheckableInputVariable:
				addr := rule.Container.(addrs.AbsInputVariableInstance)
				if !addr.Module.IsRoot() {
					// failures can only be expected against checkable objects
					// in the root module. This diagnostic will be added into
					// returned set below.
					break
				}

				if diag.Severity() == tfdiags.Warning {
					// Warnings don't count as errors. This diagnostic will be
					// added into the returned set below.
					break
				}
				if expectedFailures.Has(addr.Variable) {
					// Then this failure is expected! Mark the original map as
					// having found a failure and swallow this error by
					// continuing and not adding it into the returned set of
					// diagnostics.
					expectedFailures.Put(addr.Variable, true)
					continue
				}

				// Otherwise, this isn't an expected failure so just fall out
				// and add it into the returned set of diagnostics below.

			case addrs.CheckableResource:
				addr := rule.Container.(addrs.AbsResourceInstance)
				if !addr.Module.IsRoot() {
					// failures can only be expected against checkable objects
					// in the root module. This diagnostic will be added into
					// returned set below.
					break
				}

				if diag.Severity() == tfdiags.Warning {
					// Warnings don't count as errors. This diagnostic will be
					// added into the returned set below.
					break
				}

				if expectedFailures.Has(addr.Resource) {
					// Then this failure is expected! Mark the original map as
					// having found a failure and swallow this error by
					// continuing and not adding it into the returned set of
					// diagnostics.
					expectedFailures.Put(addr.Resource, true)
					continue
				}

				if expectedFailures.Has(addr.Resource.Resource) {
					// We can also blanket expect failures in all instances for
					// a resource so we check for that here as well.
					expectedFailures.Put(addr.Resource.Resource, true)
					continue
				}

				// Otherwise, this isn't an expected failure so just fall out
				// and add it into the returned set of diagnostics below.

			case addrs.CheckableCheck:
				addr := rule.Container.(addrs.AbsCheck)

				// Check blocks are a bit more difficult than the others. Check
				// block diagnostics could be from a nested data block, or
				// from a failed assertion, and have all been marked as just
				// warning severity.
				//
				// For diagnostics from failed assertions, we want to check if
				// it was expected and skip it if it was. But if it wasn't
				// expected we want to upgrade the diagnostic from a warning
				// into an error so the test case will fail overall.
				//
				// For diagnostics from nested data blocks, we have two
				// categories of diagnostics. First, diagnostics that were
				// originally errors and we mapped into warnings. Second,
				// diagnostics that were originally warnings and stayed that
				// way. For the first case, we want to turn these back to errors
				// and use them as part of the expected failures functionality.
				// The second case should remain as warnings and be ignored by
				// the expected failures functionality.
				//
				// Note, as well that we still want to upgrade failed checks
				// from child modules into errors, so in the other branches we
				// just do a simple blanket skip off all diagnostics not
				// from the root module. We're more selective here, only
				// diagnostics from the root module are considered for the
				// expect failures functionality but we do also upgrade
				// diagnostics from child modules back into errors.

				if rule.Type == addrs.CheckAssertion {
					// Then this diagnostic is from a check block assertion, it
					// is something we want to treat as an error even though it
					// is actually claiming to be a warning.

					if addr.Module.IsRoot() && expectedFailures.Has(addr.Check) {
						// Then this failure is expected! Mark the original map as
						// having found a failure and continue.
						expectedFailures.Put(addr.Check, true)
						continue
					}

					// Otherwise, let's package this up as an error and move on.
					diags = diags.Append(tfdiags.Override(diag, tfdiags.Error, nil))
					continue
				} else if rule.Type == addrs.CheckDataResource {
					// Then the diagnostic we have was actually overridden so
					// let's get back to the original.
					original := tfdiags.UndoOverride(diag)

					// This diagnostic originated from a scoped data source.
					if addr.Module.IsRoot() && original.Severity() == tfdiags.Error {
						// Okay, we have a genuine error from the root module,
						// so we can now check if we want to ignore it or not.
						if expectedFailures.Has(addr.Check) {
							// Then this failure is expected! Mark the original map as
							// having found a failure and continue.
							expectedFailures.Put(addr.Check, true)
							continue
						}
					}

					// In all other cases, we want to add the original error
					// into the set we return to the testing framework and move
					// onto the next one.
					diags = diags.Append(original)
					continue
				} else {
					panic("invalid CheckType: " + rule.Type.String())
				}
			default:
				panic("unrecognized CheckableKind: " + rule.Container.CheckableKind().String())
			}
		}

		// If we get here, then we're not modifying the original diagnostic at
		// all. We just want the testing framework to treat it as normal.
		diags = diags.Append(diag)
	}

	// Okay, we've checked all our diagnostics to see if any were expected.
	// Now, let's make sure that all the checkable objects we expected to fail
	// actually did!

	for _, elem := range expectedFailures.Elems {
		addr := elem.Key
		failed := elem.Value

		if !failed {
			// Then we expected a failure, and it did not occur. Add it to the
			// diagnostics.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing expected failure",
				Detail:   fmt.Sprintf("The checkable object, %s, was expected to report an error but did not.", addr.String()),
				Subject:  sourceRanges.Get(addr).ToHCL().Ptr(),
				Extra:    missingExpectedFailure(true),
			})
		}
	}

	return diags
}

// DiagnosticExtraFromMissingExpectedFailure provides an interface for diagnostic ExtraInfo to
// denote that a diagnostic was generated as a result of a missing expected failure.
type DiagnosticExtraFromMissingExpectedFailure interface {
	DiagnosticFromMissingExpectedFailure() bool
}

// DiagnosticFromMissingExpectedFailure checks if the provided diagnostic
// is a result of a missing expected failure.
func DiagnosticFromMissingExpectedFailure(diag tfdiags.Diagnostic) bool {
	maybe := tfdiags.ExtraInfo[DiagnosticExtraFromMissingExpectedFailure](diag)
	if maybe == nil {
		return false
	}
	return maybe.DiagnosticFromMissingExpectedFailure()
}

type missingExpectedFailure bool

func (missingExpectedFailure) DiagnosticFromMissingExpectedFailure() bool {
	return true
}
