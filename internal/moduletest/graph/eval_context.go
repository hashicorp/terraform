// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/moduletest"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// EvalContext is a container for context relating to the evaluation of a
// particular .tftest.hcl file.
// This context is used to track the various values that are available to the
// test suite, both from the test suite itself and from the results of the runs
// within the suite.
// The struct provides concurrency-safe access to the various maps it contains.
type EvalContext struct {
	VariableCaches *hcltest.VariableCaches

	// runOutputs is a mapping from run addresses to cty object values
	// representing the collected output values from the module under test.
	//
	// This is used to allow run blocks to refer back to the output values of
	// previous run blocks. It is passed into the Evaluate functions that
	// validate the test assertions, and used when calculating values for
	// variables within run blocks.
	runOutputs  map[addrs.Run]cty.Value
	outputsLock sync.Mutex

	// configProviders is a cache of config keys mapped to all the providers
	// referenced by the given config.
	//
	// The config keys are globally unique across an entire test suite, so we
	// store this at the suite runner level to get maximum efficiency.
	configProviders map[string]map[string]bool
	providersLock   sync.Mutex

	// FileStates is a mapping of module keys to it's last applied state
	// file.
	//
	// This is used to clean up the infrastructure created during the test after
	// the test has finished.
	FileStates map[string]*TestFileState
	stateLock  sync.Mutex

	// cancelContext and stopContext can be used to terminate the evaluation of the
	// test suite when a cancellation or stop signal is received.
	// cancelFunc and stopFunc are the corresponding functions to call to signal
	// the termination.
	cancelContext context.Context
	cancelFunc    context.CancelFunc
	stopContext   context.Context
	stopFunc      context.CancelFunc

	renderer views.Test
	verbose  bool
}

type EvalContextOpts struct {
	Verbose   bool
	Render    views.Test
	CancelCtx context.Context
	StopCtx   context.Context
}

// NewEvalContext constructs a new graph evaluation context for use in
// evaluating the runs within a test suite.
// The context is initialized with the provided cancel and stop contexts, and
// these contexts can be used from external commands to signal the termination of the test suite.
func NewEvalContext(opts *EvalContextOpts) *EvalContext {
	cancelCtx, cancel := context.WithCancel(opts.CancelCtx)
	stopCtx, stop := context.WithCancel(opts.StopCtx)
	return &EvalContext{
		runOutputs:      make(map[addrs.Run]cty.Value),
		outputsLock:     sync.Mutex{},
		configProviders: make(map[string]map[string]bool),
		providersLock:   sync.Mutex{},
		FileStates:      make(map[string]*TestFileState),
		stateLock:       sync.Mutex{},
		VariableCaches:  hcltest.NewVariableCaches(),
		cancelContext:   cancelCtx,
		cancelFunc:      cancel,
		stopContext:     stopCtx,
		stopFunc:        stop,
		verbose:         opts.Verbose,
		renderer:        opts.Render,
	}
}

// Renderer returns the renderer for the test suite.
func (ec *EvalContext) Renderer() views.Test {
	return ec.renderer
}

// Cancel signals to the runs in the test suite that they should stop evaluating
// the test suite, and return immediately.
func (ec *EvalContext) Cancel() {
	ec.cancelFunc()
}

// Cancelled returns true if the context has been stopped. The default cause
// of the error is context.Canceled.
func (ec *EvalContext) Cancelled() bool {
	return ec.cancelContext.Err() != nil
}

// Stop signals to the runs in the test suite that they should stop evaluating
// the test suite, and just skip.
func (ec *EvalContext) Stop() {
	ec.stopFunc()
}

func (ec *EvalContext) Stopped() bool {
	return ec.stopContext.Err() != nil
}

// Verbose returns true if the context is in verbose mode.
func (ec *EvalContext) Verbose() bool {
	return ec.verbose
}

// EvaluateRun processes the assertions inside the provided configs.TestRun against
// the run results, returning a status, an object value representing the output
// values from the module under test, and diagnostics describing any problems.
//
// extraVariableVals, if provided, overlays the input variables that are
// already available in resultScope in case there are additional input
// variables that were defined only for use in the test suite. Any variable
// not defined in extraVariableVals will be evaluated through resultScope instead.
func (ec *EvalContext) EvaluateRun(run *moduletest.Run, resultScope *lang.Scope, extraVariableVals terraform.InputValues) (moduletest.Status, cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if run.ModuleConfig == nil {
		// This should never happen, but if it does, we can't evaluate the run
		return moduletest.Error, cty.NilVal, tfdiags.Diagnostics{}
	}

	mod := run.ModuleConfig.Module
	// We need a derived evaluation scope that also supports referring to
	// the prior run output values using the "run.NAME" syntax.
	evalData := &evaluationData{
		ctx:       ec,
		module:    mod,
		current:   resultScope.Data,
		extraVars: extraVariableVals,
	}
	scope := &lang.Scope{
		Data:          evalData,
		ParseRef:      addrs.ParseRefFromTestingScope,
		SourceAddr:    resultScope.SourceAddr,
		BaseDir:       resultScope.BaseDir,
		PureOnly:      resultScope.PureOnly,
		PlanTimestamp: resultScope.PlanTimestamp,
		ExternalFuncs: resultScope.ExternalFuncs,
	}

	log.Printf("[TRACE] EvalContext.Evaluate for %s", run.Addr())

	// We're going to assume the run has passed, and then if anything fails this
	// value will be updated.
	status := run.Status.Merge(moduletest.Pass)

	// Now validate all the assertions within this run block.
	for i, rule := range run.Config.CheckRules {
		var ruleDiags tfdiags.Diagnostics

		refs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, rule.Condition)
		ruleDiags = ruleDiags.Append(moreDiags)
		moreRefs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, rule.ErrorMessage)
		ruleDiags = ruleDiags.Append(moreDiags)
		refs = append(refs, moreRefs...)

		// We want to emit diagnostics if users are using ephemeral resources in their checks
		// as they are not supported since they are closed before this is evaluated.
		// We do not remove the diagnostic about the ephemeral resource being closed already as it
		// might be useful to the user.
		ruleDiags = ruleDiags.Append(diagsForEphemeralResources(refs))

		hclCtx, moreDiags := scope.EvalContext(refs)
		ruleDiags = ruleDiags.Append(moreDiags)
		if moreDiags.HasErrors() {
			// if we can't evaluate the context properly, we can't evaulate the rule
			// we add the diagnostics to the main diags and continue to the next rule
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s is invalid, could not evalaute the context, so cannot evaluate it", i, run.Addr())
			status = status.Merge(moduletest.Error)
			diags = diags.Append(ruleDiags)
			continue
		}

		errorMessage, moreDiags := lang.EvalCheckErrorMessage(rule.ErrorMessage, hclCtx, nil)
		ruleDiags = ruleDiags.Append(moreDiags)

		runVal, hclDiags := rule.Condition.Value(hclCtx)
		ruleDiags = ruleDiags.Append(hclDiags)

		diags = diags.Append(ruleDiags)
		if ruleDiags.HasErrors() {
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s is invalid, so cannot evaluate it", i, run.Addr())
			status = status.Merge(moduletest.Error)
			continue
		}

		if runVal.IsNull() {
			status = status.Merge(moduletest.Error)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid condition run",
				Detail:      "Condition expression must return either true or false, not null.",
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has null condition result", i, run.Addr())
			continue
		}

		if !runVal.IsKnown() {
			status = status.Merge(moduletest.Error)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Unknown condition value",
				Detail:      "Condition expression could not be evaluated at this time. This means you have executed a `run` block with `command = plan` and one of the values your condition depended on is not known until after the plan has been applied. Either remove this value from your condition, or execute an `apply` command from this `run` block. Alternatively, if there is an override for this value, you can make it available during the plan phase by setting `override_during = plan` in the `override_` block.",
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has unknown condition result", i, run.Addr())
			continue
		}

		var err error
		if runVal, err = convert.Convert(runVal, cty.Bool); err != nil {
			status = status.Merge(moduletest.Error)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid condition run",
				Detail:      fmt.Sprintf("Invalid condition run value: %s.", tfdiags.FormatError(err)),
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
			})
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has non-boolean condition result", i, run.Addr())
			continue
		}

		// If the runVal refers to any sensitive values, then we'll have a
		// sensitive mark on the resulting value.
		runVal, _ = runVal.Unmark()

		if runVal.False() {
			log.Printf("[TRACE] EvalContext.Evaluate: test assertion failed for %s assertion %d", run.Addr(), i)
			status = status.Merge(moduletest.Fail)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Test assertion failed",
				Detail:      errorMessage,
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
				// Make the ephemerality visible
				Extra: terraform.DiagnosticCausedByEphemeral(true),
			})
			continue
		} else {
			log.Printf("[TRACE] EvalContext.Evaluate: test assertion succeeded for %s assertion %d", run.Addr(), i)
		}
	}

	// Our result includes an object representing all of the output values
	// from the module we've just tested, which will then be available in
	// any subsequent test cases in the same test suite.
	outputVals := make(map[string]cty.Value, len(mod.Outputs))
	runRng := tfdiags.SourceRangeFromHCL(run.Config.DeclRange)
	for _, oc := range mod.Outputs {
		addr := oc.Addr()
		v, moreDiags := scope.Data.GetOutput(addr, runRng)
		diags = diags.Append(moreDiags)
		if v == cty.NilVal {
			v = cty.NullVal(cty.DynamicPseudoType)
		}
		outputVals[addr.Name] = v
	}

	return status, cty.ObjectVal(outputVals), diags
}

func (ec *EvalContext) SetOutput(run *moduletest.Run, output cty.Value) {
	ec.outputsLock.Lock()
	defer ec.outputsLock.Unlock()
	ec.runOutputs[run.Addr()] = output
}

func (ec *EvalContext) GetOutputs() map[addrs.Run]cty.Value {
	ec.outputsLock.Lock()
	defer ec.outputsLock.Unlock()
	outputCopy := make(map[addrs.Run]cty.Value, len(ec.runOutputs))
	for k, v := range ec.runOutputs {
		outputCopy[k] = v
	}
	return outputCopy
}

func (ec *EvalContext) GetCache(run *moduletest.Run) *hcltest.VariableCache {
	return ec.VariableCaches.GetCache(run.Name, run.ModuleConfig)
}

// ProviderExists returns true if the provider exists for the run inside the context.
func (ec *EvalContext) ProviderExists(run *moduletest.Run, key string) bool {
	ec.providersLock.Lock()
	defer ec.providersLock.Unlock()
	runProviders, ok := ec.configProviders[run.GetModuleConfigID()]
	if !ok {
		return false
	}

	found, ok := runProviders[key]
	return ok && found
}

func (ec *EvalContext) SetProviders(run *moduletest.Run, providers map[string]bool) {
	ec.providersLock.Lock()
	defer ec.providersLock.Unlock()
	ec.configProviders[run.GetModuleConfigID()] = providers
}

func diagsForEphemeralResources(refs []*addrs.Reference) (diags tfdiags.Diagnostics) {
	for _, ref := range refs {
		switch v := ref.Subject.(type) {
		case addrs.ResourceInstance:
			if v.Resource.Mode == addrs.EphemeralResourceMode {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Ephemeral resources cannot be asserted",
					Detail:   "Ephemeral resources are closed when the test is finished, and are not available within the test context for assertions.",
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
			}
		}
	}
	return diags
}

func (ec *EvalContext) SetFileState(key string, state *TestFileState) {
	ec.stateLock.Lock()
	defer ec.stateLock.Unlock()
	ec.FileStates[key] = &TestFileState{
		Run:   state.Run,
		State: state.State,
	}
}

func (ec *EvalContext) GetFileState(key string) *TestFileState {
	ec.stateLock.Lock()
	defer ec.stateLock.Unlock()
	return ec.FileStates[key]
}

// evaluationData augments an underlying lang.Data -- presumably resulting
// from a terraform.Context.PlanAndEval or terraform.Context.ApplyAndEval call --
// with results from prior runs that should therefore be available when
// evaluating expressions written inside a "run" block.
type evaluationData struct {
	ctx       *EvalContext
	module    *configs.Module
	current   lang.Data
	extraVars terraform.InputValues
}

var _ lang.Data = (*evaluationData)(nil)

// GetCheckBlock implements lang.Data.
func (d *evaluationData) GetCheckBlock(addr addrs.Check, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetCheckBlock(addr, rng)
}

// GetCountAttr implements lang.Data.
func (d *evaluationData) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetCountAttr(addr, rng)
}

// GetForEachAttr implements lang.Data.
func (d *evaluationData) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetForEachAttr(addr, rng)
}

// GetInputVariable implements lang.Data.
func (d *evaluationData) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	if extra, exists := d.extraVars[addr.Name]; exists {
		return extra.Value, nil
	}
	return d.current.GetInputVariable(addr, rng)
}

// GetLocalValue implements lang.Data.
func (d *evaluationData) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetLocalValue(addr, rng)
}

// GetModule implements lang.Data.
func (d *evaluationData) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetModule(addr, rng)
}

// GetOutput implements lang.Data.
func (d *evaluationData) GetOutput(addr addrs.OutputValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetOutput(addr, rng)
}

// GetPathAttr implements lang.Data.
func (d *evaluationData) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetPathAttr(addr, rng)
}

// GetResource implements lang.Data.
func (d *evaluationData) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetResource(addr, rng)
}

// GetRunBlock implements lang.Data.
func (d *evaluationData) GetRunBlock(addr addrs.Run, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret, exists := d.ctx.GetOutputs()[addr]
	if !exists {
		ret = cty.DynamicVal
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared run block",
			Detail:   fmt.Sprintf("There is no run %q declared in this test suite.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
	}
	if ret == cty.NilVal {
		// An explicit nil value indicates that the block was declared but
		// hasn't yet been visited.
		ret = cty.DynamicVal
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to unevaluated run block",
			Detail:   fmt.Sprintf("The run %q block has not yet been evaluated, so its results are not available here.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
	}
	return ret, diags
}

// GetTerraformAttr implements lang.Data.
func (d *evaluationData) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return d.current.GetTerraformAttr(addr, rng)
}

// StaticValidateReferences implements lang.Data.
func (d *evaluationData) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
	// We only handle addrs.Run directly here, with everything else delegated
	// to the underlying Data object to deal with.
	var diags tfdiags.Diagnostics
	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.Run:
			diags = diags.Append(d.staticValidateRunRef(ref))
		default:
			diags = diags.Append(d.current.StaticValidateReferences([]*addrs.Reference{ref}, self, source))
		}
	}
	return diags
}

func (d *evaluationData) staticValidateRunRef(ref *addrs.Reference) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	addr := ref.Subject.(addrs.Run)
	outputs := d.ctx.GetOutputs()
	_, exists := outputs[addr]
	if !exists {
		var suggestions []string
		for altAddr := range outputs {
			suggestions = append(suggestions, altAddr.Name)
		}
		sort.Strings(suggestions)
		suggestion := didyoumean.NameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}
		// A totally absent priorVals means that there is no run block with
		// the given name at all. If it was declared but hasn't yet been
		// evaluated then it would have an entry set to cty.NilVal.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared run block",
			Detail:   fmt.Sprintf("There is no run %q declared in this test suite.%s", addr.Name, suggestion),
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		})
	}

	return diags
}
