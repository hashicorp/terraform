// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	teststates "github.com/hashicorp/terraform/internal/moduletest/states"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
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
	// unparsedVariables and parsedVariables are the values for the variables
	// required by this test file. The parsedVariables will be populated as the
	// test graph is executed, while the unparsedVariables will be lazily
	// evaluated by each run block that needs them.
	unparsedVariables map[string]backendrun.UnparsedVariableValue
	parsedVariables   terraform.InputValues
	variableStatus    map[string]moduletest.Status
	variablesLock     sync.Mutex

	// runBlocks caches all the known run blocks that this EvalContext manages.
	runBlocks   map[string]*moduletest.Run
	outputsLock sync.Mutex

	providers      map[addrs.RootProviderConfig]providers.Interface
	providerStatus map[addrs.RootProviderConfig]moduletest.Status
	providersLock  sync.Mutex

	// FileStates is a mapping of module keys to it's last applied state
	// file. This is tracked and returned to log state files of ongoing test
	// operations.
	FileStates map[string]*teststates.TestRunState
	stateLock  sync.Mutex

	// cancelContext and stopContext can be used to terminate the evaluation of the
	// test suite when a cancellation or stop signal is received.
	// cancelFunc and stopFunc are the corresponding functions to call to signal
	// the termination.
	cancelContext context.Context
	cancelFunc    context.CancelFunc
	stopContext   context.Context
	stopFunc      context.CancelFunc
	config        *configs.Config
	renderer      views.Test
	verbose       bool

	// mode and repair affect the behaviour of the cleanup process of the graph.
	//
	// in cleanup mode, the tests will actually be skipped and the cleanup nodes
	// are executed immediately. Normally, the skip_cleanup attributes will
	// be skipped in cleanup mode with all states being destroyed completely.
	//
	// in repair mode, the skip_cleanup attributes are still respected. this
	// means only states that were left behind due to an error will be
	// destroyed.
	mode moduletest.CommandMode

	deferralAllowed bool
	evalSem         terraform.Semaphore

	// repair is true if the test suite is being run in cleanup repair mode.
	// It is only set when in test cleanup mode.
	repair bool

	overrides    map[string]*mocking.Overrides
	overrideLock sync.Mutex
}

type EvalContextOpts struct {
	Verbose           bool
	Repair            bool
	Render            views.Test
	CancelCtx         context.Context
	StopCtx           context.Context
	UnparsedVariables map[string]backendrun.UnparsedVariableValue
	Config            *configs.Config
	FileStates        map[string]*teststates.TestRunState
	Concurrency       int
	DeferralAllowed   bool
	Mode              moduletest.CommandMode
}

// NewEvalContext constructs a new graph evaluation context for use in
// evaluating the runs within a test suite.
// The context is initialized with the provided cancel and stop contexts, and
// these contexts can be used from external commands to signal the termination of the test suite.
func NewEvalContext(opts EvalContextOpts) *EvalContext {
	cancelCtx, cancel := context.WithCancel(opts.CancelCtx)
	stopCtx, stop := context.WithCancel(opts.StopCtx)
	return &EvalContext{
		unparsedVariables: opts.UnparsedVariables,
		parsedVariables:   make(terraform.InputValues),
		variableStatus:    make(map[string]moduletest.Status),
		variablesLock:     sync.Mutex{},
		runBlocks:         make(map[string]*moduletest.Run),
		outputsLock:       sync.Mutex{},
		providers:         make(map[addrs.RootProviderConfig]providers.Interface),
		providerStatus:    make(map[addrs.RootProviderConfig]moduletest.Status),
		providersLock:     sync.Mutex{},
		FileStates:        opts.FileStates,
		stateLock:         sync.Mutex{},
		cancelContext:     cancelCtx,
		cancelFunc:        cancel,
		stopContext:       stopCtx,
		stopFunc:          stop,
		config:            opts.Config,
		verbose:           opts.Verbose,
		repair:            opts.Repair,
		renderer:          opts.Render,
		mode:              opts.Mode,
		deferralAllowed:   opts.DeferralAllowed,
		evalSem:           terraform.NewSemaphore(opts.Concurrency),
		overrides:         make(map[string]*mocking.Overrides),
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

func (ec *EvalContext) HclContext(references []*addrs.Reference) (*hcl.EvalContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	runs := make(map[string]cty.Value)
	vars := make(map[string]cty.Value)

	for _, reference := range references {
		switch subject := reference.Subject.(type) {
		case addrs.Run:
			run, ok := ec.GetOutput(subject.Name)
			if !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to unknown run block",
					Detail:   fmt.Sprintf("The run block %q does not exist within this test file.", subject.Name),
					Subject:  reference.SourceRange.ToHCL().Ptr(),
				})
				continue
			}
			runs[subject.Name] = run

			value, valueDiags := reference.Remaining.TraverseRel(run)
			diags = diags.Append(valueDiags)
			if valueDiags.HasErrors() {
				continue
			}

			if !value.IsWhollyKnown() {
				// This is not valid, we cannot allow users to pass unknown
				// values into references within the test file. There's just
				// going to be difficult and confusing errors later if this
				// happens.
				//
				// When reporting this we assume that it's happened because
				// the prior run was a plan-only run and that some of its
				// output values were not known. If this arises for a
				// run that performed a full apply then this is a bug in
				// Terraform's modules runtime, because unknown output
				// values should not be possible in that case.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to unknown value",
					Detail:   fmt.Sprintf("The value for %s is unknown. Run block %q is executing a \"plan\" operation, and the specified output value is only known after apply.", reference.DisplayString(), subject.Name),
					Subject:  reference.SourceRange.ToHCL().Ptr(),
				})
				continue
			}

		case addrs.InputVariable:
			if variable, ok := ec.GetVariable(subject.Name); ok {
				vars[subject.Name] = variable.Value
				continue
			}

			if variable, moreDiags := ec.EvaluateUnparsedVariableDeprecated(subject.Name, reference); variable != nil {
				diags = diags.Append(moreDiags)
				vars[subject.Name] = variable.Value
				continue
			}

			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to unavailable variable",
				Detail:   fmt.Sprintf("The input variable %q does not exist within this test file.", subject.Name),
				Subject:  reference.SourceRange.ToHCL().Ptr(),
			})
			continue

		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   "You can only reference run blocks and variables from within Terraform Test files.",
				Subject:  reference.SourceRange.ToHCL().Ptr(),
			})
		}
	}

	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"run": cty.ObjectVal(runs),
			"var": cty.ObjectVal(vars),
		},
		Functions: lang.TestingFunctions(),
	}, diags
}

// EvaluateRun processes the assertions inside the provided configs.TestRun against
// the run results, returning a status, an object value representing the output
// values from the module under test, and diagnostics describing any problems.
//
// extraVariableVals, if provided, overlays the input variables that are
// already available in resultScope in case there are additional input
// variables that were defined only for use in the test suite. Any variable
// not defined in extraVariableVals will be evaluated through resultScope instead.
func (ec *EvalContext) EvaluateRun(run *configs.TestRun, module *configs.Module, resultScope *lang.Scope, extraVariableVals terraform.InputValues) (moduletest.Status, cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// We need a derived evaluation scope that also supports referring to
	// the prior run output values using the "run.NAME" syntax.
	evalData := &evaluationData{
		ctx:       ec,
		module:    module,
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

	log.Printf("[TRACE] EvalContext.Evaluate for %s", run.Name)

	// We're going to assume the run has passed, and then if anything fails this
	// value will be updated.
	status := moduletest.Pass

	// Now validate all the assertions within this run block.
	for i, rule := range run.CheckRules {
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
			// if we can't evaluate the context properly, we can't evaluate the rule
			// we add the diagnostics to the main diags and continue to the next rule
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s is invalid, could not evalaute the context, so cannot evaluate it", i, run.Name)
			status = status.Merge(moduletest.Error)
			diags = diags.Append(ruleDiags)
			continue
		}

		errorMessage, moreDiags := lang.EvalCheckErrorMessage(rule.ErrorMessage, hclCtx, nil)
		ruleDiags = ruleDiags.Append(moreDiags)

		errorMessage, _ = errorMessage.Unmark()
		errorMessageStr := strings.TrimSpace(errorMessage.AsString())

		runVal, hclDiags := rule.Condition.Value(hclCtx)
		ruleDiags = ruleDiags.Append(hclDiags)

		diags = diags.Append(ruleDiags)
		if ruleDiags.HasErrors() {
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s is invalid, so cannot evaluate it", i, run.Name)
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
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has null condition result", i, run.Name)
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
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has unknown condition result", i, run.Name)
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
			log.Printf("[TRACE] EvalContext.Evaluate: check rule %d for %s has non-boolean condition result", i, run.Name)
			continue
		}

		// If the runVal refers to any sensitive values, then we'll have a
		// sensitive mark on the resulting value.
		runVal, _ = runVal.Unmark()

		if runVal.False() {
			log.Printf("[TRACE] EvalContext.Evaluate: test assertion failed for %s assertion %d", run.Name, i)
			status = status.Merge(moduletest.Fail)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Test assertion failed",
				Detail:      errorMessageStr,
				Subject:     rule.Condition.Range().Ptr(),
				Expression:  rule.Condition,
				EvalContext: hclCtx,
				// Diagnostic can be identified as originating from a failing test assertion.
				// Also, values that are ephemeral, sensitive, or unknown are replaced with
				// redacted values in renderings of the diagnostic.
				Extra: DiagnosticCausedByTestFailure{Verbose: ec.verbose},
			})
			continue
		} else {
			log.Printf("[TRACE] EvalContext.Evaluate: test assertion succeeded for %s assertion %d", run.Name, i)
		}
	}

	// Our result includes an object representing all of the output values
	// from the module we've just tested, which will then be available in
	// any subsequent test cases in the same test suite.
	outputVals := make(map[string]cty.Value, len(module.Outputs))
	runRng := tfdiags.SourceRangeFromHCL(run.DeclRange)
	for _, oc := range module.Outputs {
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

// EvaluateUnparsedVariable accepts a variable name and a variable definition
// and checks if we have external unparsed variables that match the given
// configuration. If no variable was provided, we'll return a nil
// input value.
func (ec *EvalContext) EvaluateUnparsedVariable(name string, config *configs.Variable) (*terraform.InputValue, tfdiags.Diagnostics) {
	variable, exists := ec.unparsedVariables[name]
	if !exists {
		return nil, nil
	}

	value, diags := variable.ParseVariableValue(config.ParsingMode)
	if diags.HasErrors() {
		value = &terraform.InputValue{
			Value: cty.DynamicVal,
		}
	}

	return value, diags
}

// EvaluateUnparsedVariableDeprecated accepts a variable name without a variable
// definition and attempts to parse it.
//
// This function represents deprecated functionality within the testing
// framework. It is no longer valid to reference external variables without a
// definition, but we do our best here and provide a warning that this will
// become completely unsupported in the future.
func (ec *EvalContext) EvaluateUnparsedVariableDeprecated(name string, ref *addrs.Reference) (*terraform.InputValue, tfdiags.Diagnostics) {
	variable, exists := ec.unparsedVariables[name]
	if !exists {
		return nil, nil
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Variable referenced without definition",
		Detail:   fmt.Sprintf("Variable %q was referenced without providing a definition. Referencing undefined variables within Terraform Test files is deprecated, please add a `variable` block into the relevant test file to provide a definition for the variable. This will become required in future versions of Terraform.", name),
		Subject:  ref.SourceRange.ToHCL().Ptr(),
	})

	// For backwards-compatibility reasons we do also have to support trying
	// to parse the global variables without a configuration. We introduced the
	// file-level variable definitions later, and users were already using
	// global variables so we do need to keep supporting this use case.

	// Otherwise, we have no configuration so we're going to try both parsing
	// modes.

	value, moreDiags := variable.ParseVariableValue(configs.VariableParseHCL)
	diags = diags.Append(moreDiags)
	if !moreDiags.HasErrors() {
		// then good! we can just return these values directly.
		return value, diags
	}

	// otherwise, we'll try the other one.

	value, moreDiags = variable.ParseVariableValue(configs.VariableParseLiteral)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		// as usual make sure we still provide something for this value.
		value = &terraform.InputValue{
			Value: cty.DynamicVal,
		}
	}
	return value, diags
}

func (ec *EvalContext) SetVariable(name string, val *terraform.InputValue) {
	ec.variablesLock.Lock()
	defer ec.variablesLock.Unlock()

	ec.parsedVariables[name] = val
}

func (ec *EvalContext) GetVariable(name string) (*terraform.InputValue, bool) {
	ec.variablesLock.Lock()
	defer ec.variablesLock.Unlock()

	variable, ok := ec.parsedVariables[name]
	return variable, ok
}

func (ec *EvalContext) SetVariableStatus(address string, status moduletest.Status) {
	ec.variablesLock.Lock()
	defer ec.variablesLock.Unlock()
	ec.variableStatus[address] = status
}

func (ec *EvalContext) AddRunBlock(run *moduletest.Run) {
	ec.outputsLock.Lock()
	defer ec.outputsLock.Unlock()
	ec.runBlocks[run.Name] = run
}

func (ec *EvalContext) GetOutput(name string) (cty.Value, bool) {
	ec.outputsLock.Lock()
	defer ec.outputsLock.Unlock()
	output, ok := ec.runBlocks[name]
	if !ok {
		return cty.NilVal, false
	}
	return output.Outputs, true
}

func (ec *EvalContext) ProviderForConfigAddr(addr addrs.LocalProviderConfig) addrs.Provider {
	return ec.config.ProviderForConfigAddr(addr)
}

func (ec *EvalContext) LocalNameForProvider(addr addrs.RootProviderConfig) string {
	return ec.config.Module.LocalNameForProvider(addr.Provider)
}

func (ec *EvalContext) GetProvider(addr addrs.RootProviderConfig) (providers.Interface, bool) {
	ec.providersLock.Lock()
	defer ec.providersLock.Unlock()
	provider, ok := ec.providers[addr]
	return provider, ok
}

func (ec *EvalContext) SetProvider(addr addrs.RootProviderConfig, provider providers.Interface) {
	ec.providersLock.Lock()
	defer ec.providersLock.Unlock()
	ec.providers[addr] = provider
}

func (ec *EvalContext) SetProviderStatus(addr addrs.RootProviderConfig, status moduletest.Status) {
	ec.providersLock.Lock()
	defer ec.providersLock.Unlock()
	ec.providerStatus[addr] = status
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

func (ec *EvalContext) SetFileState(key string, run *moduletest.Run, state *states.State, reason teststates.StateReason) {
	ec.stateLock.Lock()
	defer ec.stateLock.Unlock()

	current := ec.getState(key)

	// Whatever happens we're going to record the latest state for this key.
	current.State = state
	current.Manifest.Reason = reason

	if run.Config.SkipCleanup {
		// if skip cleanup is set on the run block, we're going to track it
		// as the thing to target regardless of what else might be true.
		current.Run = run

		// we'll mark the state as being restored to the current run block
		// if (a) we're not in cleanup mode (meaning everything should be
		// destroyed) or (b) we are in cleanup mode but with the repair flag
		// which means that only errored states should be destroyed.
		current.RestoreState = ec.mode != moduletest.CleanupMode || ec.repair
	} else if !current.RestoreState {
		// otherwise, only set the new run block if we haven't been told the
		// earlier run block is more relevant.
		current.Run = run
	}
}

// GetState retrieves the current state for the specified key, exactly as it
// specified within the current cache.
func (ec *EvalContext) GetState(key string) *teststates.TestRunState {
	ec.stateLock.Lock()
	defer ec.stateLock.Unlock()
	return ec.getState(key)
}

func (ec *EvalContext) getState(key string) *teststates.TestRunState {
	current := ec.FileStates[key]
	if current == nil {
		// this shouldn't happen, all the states must be initialised prior to
		// the evaluation context being created.
		//
		// panic here, where the origin of the bug is instead of returning a
		// null state to panic later.
		panic("null state found in test execution")
	}
	return current
}

// LoadState returns the correct state for the specified run block. This differs
// from GetState in that it will load the state from any remote backend
// specified within the run block rather than simply retrieve the cached state
// (which might be empty for a run block with a backend if it hasn't executed
// yet).
func (ec *EvalContext) LoadState(run *configs.TestRun) (*states.State, error) {
	ec.stateLock.Lock()
	defer ec.stateLock.Unlock()

	current := ec.getState(run.StateKey)

	if run.Backend != nil {
		// Then we'll load the state from the backend instead of just using
		// whatever was in the state.

		stmgr, sDiags := current.Backend.StateMgr(backend.DefaultStateName)
		if sDiags.HasErrors() {
			return nil, sDiags.Err()
		}

		if err := stmgr.RefreshState(); err != nil {
			return nil, err
		}

		return stmgr.State(), nil
	}

	return current.State, nil
}

// ReferencesCompleted returns true if all the listed references were actually
// executed successfully. This allows nodes in the graph to decide if they
// should execute or not based on the status of their references.
func (ec *EvalContext) ReferencesCompleted(refs []*addrs.Reference) bool {
	for _, ref := range refs {
		switch ref := ref.Subject.(type) {
		case addrs.Run:
			ec.outputsLock.Lock()
			if run, ok := ec.runBlocks[ref.Name]; ok {
				if run.Status != moduletest.Pass && run.Status != moduletest.Fail {
					ec.outputsLock.Unlock()

					// see also prior runs completed

					return false
				}
			}
			ec.outputsLock.Unlock()
		case addrs.InputVariable:
			ec.variablesLock.Lock()
			if vStatus, ok := ec.variableStatus[ref.Name]; ok && (vStatus == moduletest.Skip || vStatus == moduletest.Error) {
				ec.variablesLock.Unlock()
				return false
			}
			ec.variablesLock.Unlock()
		}
	}
	return true
}

// ProvidersCompleted ensures that all required providers were properly
// initialised.
func (ec *EvalContext) ProvidersCompleted(providers map[addrs.RootProviderConfig]providers.Interface) bool {
	ec.providersLock.Lock()
	defer ec.providersLock.Unlock()

	for provider := range providers {
		if status, ok := ec.providerStatus[provider]; ok {
			if status == moduletest.Skip || status == moduletest.Error {
				return false
			}
		}
	}
	return true
}

// PriorRunsCompleted checks a list of run blocks against our internal log of
// completed run blocks and makes sure that any that do exist successfully
// executed to completion.
//
// Note that run blocks that are not in the list indicate a bad reference,
// which we ignore here. This is actually the problem of the caller to identify
// and error.
func (ec *EvalContext) PriorRunsCompleted(runs map[string]*moduletest.Run) bool {
	ec.outputsLock.Lock()
	defer ec.outputsLock.Unlock()

	for name := range runs {
		if run, ok := ec.runBlocks[name]; ok {
			if run.Status != moduletest.Pass && run.Status != moduletest.Fail {

				// pass and fail indicate the run block still executed the plan
				// or apply operate and wrote outputs. fail means the
				// post-execution checks failed, but we still had data to check.
				// this is in contrast to pending, skip, or error which indicate
				// that we never even wrote data for this run block.

				return false
			}
		}
	}
	return true
}

func (ec *EvalContext) SetOverrides(run *moduletest.Run, overrides *mocking.Overrides) {
	ec.overrideLock.Lock()
	defer ec.overrideLock.Unlock()
	ec.overrides[run.Name] = overrides
}

func (ec *EvalContext) GetOverrides(runName string) *mocking.Overrides {
	ec.overrideLock.Lock()
	defer ec.overrideLock.Unlock()
	return ec.overrides[runName]
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
	ret, exists := d.ctx.GetOutput(addr.Name)
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
	d.ctx.outputsLock.Lock()
	defer d.ctx.outputsLock.Unlock()

	var diags tfdiags.Diagnostics

	addr := ref.Subject.(addrs.Run)

	if _, exists := d.ctx.runBlocks[addr.Name]; !exists {
		var suggestions []string
		for altAddr := range d.ctx.runBlocks {
			suggestions = append(suggestions, altAddr)
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
