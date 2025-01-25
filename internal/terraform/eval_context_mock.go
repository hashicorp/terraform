// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/experiments"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/resources/ephemeral"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MockEvalContext is a mock version of EvalContext that can be used
// for tests.
type MockEvalContext struct {
	StopCtxCalled bool
	StopCtxValue  context.Context

	HookCalled bool
	HookHook   Hook
	HookError  error

	InputCalled bool
	InputInput  UIInput

	InitProviderCalled   bool
	InitProviderType     string
	InitProviderAddr     addrs.AbsProviderConfig
	InitProviderProvider providers.Interface
	InitProviderError    error

	ProviderCalled   bool
	ProviderAddr     addrs.AbsProviderConfig
	ProviderProvider providers.Interface

	ProviderSchemaCalled bool
	ProviderSchemaAddr   addrs.AbsProviderConfig
	ProviderSchemaSchema providers.ProviderSchema
	ProviderSchemaError  error

	CloseProviderCalled   bool
	CloseProviderAddr     addrs.AbsProviderConfig
	CloseProviderProvider providers.Interface

	ProviderInputCalled bool
	ProviderInputAddr   addrs.AbsProviderConfig
	ProviderInputValues map[string]cty.Value

	SetProviderInputCalled bool
	SetProviderInputAddr   addrs.AbsProviderConfig
	SetProviderInputValues map[string]cty.Value

	ConfigureProviderFn func(
		addr addrs.AbsProviderConfig,
		cfg cty.Value) tfdiags.Diagnostics // overrides the other values below, if set
	ConfigureProviderCalled bool
	ConfigureProviderAddr   addrs.AbsProviderConfig
	ConfigureProviderConfig cty.Value
	ConfigureProviderDiags  tfdiags.Diagnostics

	ProvisionerCalled      bool
	ProvisionerName        string
	ProvisionerProvisioner provisioners.Interface

	ProvisionerSchemaCalled bool
	ProvisionerSchemaName   string
	ProvisionerSchemaSchema *configschema.Block
	ProvisionerSchemaError  error

	ClosePluginsCalled bool

	EvaluateBlockCalled     bool
	EvaluateBlockBody       hcl.Body
	EvaluateBlockSchema     *configschema.Block
	EvaluateBlockSelf       addrs.Referenceable
	EvaluateBlockKeyData    InstanceKeyEvalData
	EvaluateBlockResultFunc func(
		body hcl.Body,
		schema *configschema.Block,
		self addrs.Referenceable,
		keyData InstanceKeyEvalData,
	) (cty.Value, hcl.Body, tfdiags.Diagnostics) // overrides the other values below, if set
	EvaluateBlockResult       cty.Value
	EvaluateBlockExpandedBody hcl.Body
	EvaluateBlockDiags        tfdiags.Diagnostics

	EvaluateExprCalled     bool
	EvaluateExprExpr       hcl.Expression
	EvaluateExprWantType   cty.Type
	EvaluateExprSelf       addrs.Referenceable
	EvaluateExprResultFunc func(
		expr hcl.Expression,
		wantType cty.Type,
		self addrs.Referenceable,
	) (cty.Value, tfdiags.Diagnostics) // overrides the other values below, if set
	EvaluateExprResult cty.Value
	EvaluateExprDiags  tfdiags.Diagnostics

	EvaluationScopeCalled  bool
	EvaluationScopeSelf    addrs.Referenceable
	EvaluationScopeKeyData InstanceKeyEvalData
	EvaluationScopeScope   *lang.Scope

	PathCalled bool
	Scope      evalContextScope

	LanguageExperimentsActive experiments.Set

	NamedValuesCalled bool
	NamedValuesState  *namedvals.State

	DeferralsCalled bool
	DeferralsState  *deferring.Deferred

	ChangesCalled  bool
	ChangesChanges *plans.ChangesSync

	StateCalled bool
	StateState  *states.SyncState

	ChecksCalled bool
	ChecksState  *checks.State

	RefreshStateCalled bool
	RefreshStateState  *states.SyncState

	PrevRunStateCalled bool
	PrevRunStateState  *states.SyncState

	MoveResultsCalled  bool
	MoveResultsResults refactoring.MoveResults

	InstanceExpanderCalled   bool
	InstanceExpanderExpander *instances.Expander

	EphemeralResourcesCalled    bool
	EphemeralResourcesResources *ephemeral.Resources

	OverridesCalled bool
	OverrideValues  *mocking.Overrides

	ForgetCalled bool
	ForgetValues bool
}

// MockEvalContext implements EvalContext
var _ EvalContext = (*MockEvalContext)(nil)

func (c *MockEvalContext) StopCtx() context.Context {
	c.StopCtxCalled = true
	if c.StopCtxValue != nil {
		return c.StopCtxValue
	}
	return context.TODO()
}

func (c *MockEvalContext) Hook(fn func(Hook) (HookAction, error)) error {
	c.HookCalled = true
	if c.HookHook != nil {
		if _, err := fn(c.HookHook); err != nil {
			return err
		}
	}

	return c.HookError
}

func (c *MockEvalContext) Input() UIInput {
	c.InputCalled = true
	return c.InputInput
}

func (c *MockEvalContext) InitProvider(addr addrs.AbsProviderConfig, _ *configs.Provider) (providers.Interface, error) {
	c.InitProviderCalled = true
	c.InitProviderType = addr.String()
	c.InitProviderAddr = addr
	return c.InitProviderProvider, c.InitProviderError
}

func (c *MockEvalContext) Provider(addr addrs.AbsProviderConfig) providers.Interface {
	c.ProviderCalled = true
	c.ProviderAddr = addr
	return c.ProviderProvider
}

func (c *MockEvalContext) ProviderSchema(addr addrs.AbsProviderConfig) (providers.ProviderSchema, error) {
	c.ProviderSchemaCalled = true
	c.ProviderSchemaAddr = addr
	return c.ProviderSchemaSchema, c.ProviderSchemaError
}

func (c *MockEvalContext) CloseProvider(addr addrs.AbsProviderConfig) error {
	c.CloseProviderCalled = true
	c.CloseProviderAddr = addr
	return nil
}

func (c *MockEvalContext) ConfigureProvider(addr addrs.AbsProviderConfig, cfg cty.Value) tfdiags.Diagnostics {

	c.ConfigureProviderCalled = true
	c.ConfigureProviderAddr = addr
	c.ConfigureProviderConfig = cfg
	if c.ConfigureProviderFn != nil {
		return c.ConfigureProviderFn(addr, cfg)
	}
	return c.ConfigureProviderDiags
}

func (c *MockEvalContext) ProviderInput(addr addrs.AbsProviderConfig) map[string]cty.Value {
	c.ProviderInputCalled = true
	c.ProviderInputAddr = addr
	return c.ProviderInputValues
}

func (c *MockEvalContext) SetProviderInput(addr addrs.AbsProviderConfig, vals map[string]cty.Value) {
	c.SetProviderInputCalled = true
	c.SetProviderInputAddr = addr
	c.SetProviderInputValues = vals
}

func (c *MockEvalContext) Provisioner(n string) (provisioners.Interface, error) {
	c.ProvisionerCalled = true
	c.ProvisionerName = n
	return c.ProvisionerProvisioner, nil
}

func (c *MockEvalContext) ProvisionerSchema(n string) (*configschema.Block, error) {
	c.ProvisionerSchemaCalled = true
	c.ProvisionerSchemaName = n
	return c.ProvisionerSchemaSchema, c.ProvisionerSchemaError
}

func (c *MockEvalContext) ClosePlugins() error {
	c.ClosePluginsCalled = true
	return nil
}

func (c *MockEvalContext) EvaluateBlock(body hcl.Body, schema *configschema.Block, self addrs.Referenceable, keyData InstanceKeyEvalData) (cty.Value, hcl.Body, tfdiags.Diagnostics) {
	c.EvaluateBlockCalled = true
	c.EvaluateBlockBody = body
	c.EvaluateBlockSchema = schema
	c.EvaluateBlockSelf = self
	c.EvaluateBlockKeyData = keyData
	if c.EvaluateBlockResultFunc != nil {
		return c.EvaluateBlockResultFunc(body, schema, self, keyData)
	}
	return c.EvaluateBlockResult, c.EvaluateBlockExpandedBody, c.EvaluateBlockDiags
}

func (c *MockEvalContext) EvaluateExpr(expr hcl.Expression, wantType cty.Type, self addrs.Referenceable) (cty.Value, tfdiags.Diagnostics) {
	c.EvaluateExprCalled = true
	c.EvaluateExprExpr = expr
	c.EvaluateExprWantType = wantType
	c.EvaluateExprSelf = self
	if c.EvaluateExprResultFunc != nil {
		return c.EvaluateExprResultFunc(expr, wantType, self)
	}
	return c.EvaluateExprResult, c.EvaluateExprDiags
}

func (c *MockEvalContext) EvaluateReplaceTriggeredBy(hcl.Expression, instances.RepetitionData) (*addrs.Reference, bool, tfdiags.Diagnostics) {
	return nil, false, nil
}

// installSimpleEval is a helper to install a simple mock implementation of
// both EvaluateBlock and EvaluateExpr into the receiver.
//
// These default implementations will either evaluate the given input against
// the scope in field EvaluationScopeScope or, if it is nil, with no eval
// context at all so that only constant values may be used.
//
// This function overwrites any existing functions installed in fields
// EvaluateBlockResultFunc and EvaluateExprResultFunc.
func (c *MockEvalContext) installSimpleEval() {
	c.EvaluateBlockResultFunc = func(body hcl.Body, schema *configschema.Block, self addrs.Referenceable, keyData InstanceKeyEvalData) (cty.Value, hcl.Body, tfdiags.Diagnostics) {
		if scope := c.EvaluationScopeScope; scope != nil {
			// Fully-functional codepath.
			var diags tfdiags.Diagnostics
			body, diags = scope.ExpandBlock(body, schema)
			if diags.HasErrors() {
				return cty.DynamicVal, body, diags
			}
			val, evalDiags := c.EvaluationScopeScope.EvalBlock(body, schema)
			diags = diags.Append(evalDiags)
			if evalDiags.HasErrors() {
				return cty.DynamicVal, body, diags
			}
			return val, body, diags
		}

		// Fallback codepath supporting constant values only.
		val, hclDiags := hcldec.Decode(body, schema.DecoderSpec(), nil)
		return val, body, tfdiags.Diagnostics(nil).Append(hclDiags)
	}
	c.EvaluateExprResultFunc = func(expr hcl.Expression, wantType cty.Type, self addrs.Referenceable) (cty.Value, tfdiags.Diagnostics) {
		if scope := c.EvaluationScopeScope; scope != nil {
			// Fully-functional codepath.
			return scope.EvalExpr(expr, wantType)
		}

		// Fallback codepath supporting constant values only.
		var diags tfdiags.Diagnostics
		val, hclDiags := expr.Value(nil)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			return cty.DynamicVal, diags
		}
		var err error
		val, err = convert.Convert(val, wantType)
		if err != nil {
			diags = diags.Append(err)
			return cty.DynamicVal, diags
		}
		return val, diags
	}
}

func (c *MockEvalContext) EvaluationScope(self addrs.Referenceable, source addrs.Referenceable, keyData InstanceKeyEvalData) *lang.Scope {
	c.EvaluationScopeCalled = true
	c.EvaluationScopeSelf = self
	c.EvaluationScopeKeyData = keyData
	return c.EvaluationScopeScope
}

func (c *MockEvalContext) withScope(scope evalContextScope) EvalContext {
	newC := *c
	newC.Scope = scope
	return &newC
}

func (c *MockEvalContext) Path() addrs.ModuleInstance {
	c.PathCalled = true
	// This intentionally panics if scope isn't a module instance; callers
	// should use this only for an eval context that's working in a
	// fully-expanded module instance.
	return c.Scope.(evalContextModuleInstance).Addr
}

func (c *MockEvalContext) LanguageExperimentActive(experiment experiments.Experiment) bool {
	// This particular function uses a live data structure so that tests can
	// exercise different experiments being enabled; there is little reason
	// to directly test whether this function was called since we use this
	// function only temporarily while an experiment is active, and then
	// remove the calls once the experiment is concluded.
	return c.LanguageExperimentsActive.Has(experiment)
}

func (c *MockEvalContext) NamedValues() *namedvals.State {
	c.NamedValuesCalled = true
	return c.NamedValuesState
}

func (c *MockEvalContext) EphemeralResources() *ephemeral.Resources {
	c.EphemeralResourcesCalled = true
	return c.EphemeralResourcesResources
}

func (c *MockEvalContext) Deferrals() *deferring.Deferred {
	c.DeferralsCalled = true
	return c.DeferralsState
}

func (c *MockEvalContext) Changes() *plans.ChangesSync {
	c.ChangesCalled = true
	return c.ChangesChanges
}

func (c *MockEvalContext) State() *states.SyncState {
	c.StateCalled = true
	return c.StateState
}

func (c *MockEvalContext) Checks() *checks.State {
	c.ChecksCalled = true
	return c.ChecksState
}

func (c *MockEvalContext) RefreshState() *states.SyncState {
	c.RefreshStateCalled = true
	return c.RefreshStateState
}

func (c *MockEvalContext) PrevRunState() *states.SyncState {
	c.PrevRunStateCalled = true
	return c.PrevRunStateState
}

func (c *MockEvalContext) MoveResults() refactoring.MoveResults {
	c.MoveResultsCalled = true
	return c.MoveResultsResults
}

func (c *MockEvalContext) InstanceExpander() *instances.Expander {
	c.InstanceExpanderCalled = true
	return c.InstanceExpanderExpander
}

func (c *MockEvalContext) Overrides() *mocking.Overrides {
	c.OverridesCalled = true
	return c.OverrideValues
}

func (c *MockEvalContext) Forget() bool {
	c.ForgetCalled = true
	return c.ForgetValues
}

func (ctx *MockEvalContext) ClientCapabilities() providers.ClientCapabilities {
	return providers.ClientCapabilities{
		DeferralAllowed:            ctx.Deferrals().DeferralAllowed(),
		WriteOnlyAttributesAllowed: true,
	}
}
