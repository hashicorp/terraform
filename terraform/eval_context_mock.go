package terraform

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// MockEvalContext is a mock version of EvalContext that can be used
// for tests.
type MockEvalContext struct {
	StoppedCalled bool
	StoppedValue  <-chan struct{}

	HookCalled bool
	HookHook   Hook
	HookError  error

	InputCalled bool
	InputInput  UIInput

	InitProviderCalled   bool
	InitProviderType     string
	InitProviderAddr     addrs.ProviderConfig
	InitProviderProvider providers.Interface
	InitProviderError    error

	ProviderCalled   bool
	ProviderAddr     addrs.AbsProviderConfig
	ProviderProvider providers.Interface

	ProviderSchemaCalled bool
	ProviderSchemaAddr   addrs.AbsProviderConfig
	ProviderSchemaSchema *ProviderSchema

	CloseProviderCalled   bool
	CloseProviderAddr     addrs.ProviderConfig
	CloseProviderProvider providers.Interface

	ProviderInputCalled bool
	ProviderInputAddr   addrs.ProviderConfig
	ProviderInputValues map[string]cty.Value

	SetProviderInputCalled bool
	SetProviderInputAddr   addrs.ProviderConfig
	SetProviderInputValues map[string]cty.Value

	ConfigureProviderCalled bool
	ConfigureProviderAddr   addrs.ProviderConfig
	ConfigureProviderConfig cty.Value
	ConfigureProviderDiags  tfdiags.Diagnostics

	InitProvisionerCalled      bool
	InitProvisionerName        string
	InitProvisionerProvisioner ResourceProvisioner
	InitProvisionerError       error

	ProvisionerCalled      bool
	ProvisionerName        string
	ProvisionerProvisioner ResourceProvisioner

	ProvisionerSchemaCalled bool
	ProvisionerSchemaName   string
	ProvisionerSchemaSchema *configschema.Block

	CloseProvisionerCalled      bool
	CloseProvisionerName        string
	CloseProvisionerProvisioner ResourceProvisioner

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

	InterpolateCalled       bool
	InterpolateConfig       *config.RawConfig
	InterpolateResource     *Resource
	InterpolateConfigResult *ResourceConfig
	InterpolateError        error

	InterpolateProviderCalled       bool
	InterpolateProviderConfig       *config.ProviderConfig
	InterpolateProviderResource     *Resource
	InterpolateProviderConfigResult *ResourceConfig
	InterpolateProviderError        error

	PathCalled bool
	PathPath   addrs.ModuleInstance

	SetModuleCallArgumentsCalled bool
	SetModuleCallArgumentsModule addrs.ModuleCallInstance
	SetModuleCallArgumentsValues map[string]cty.Value

	ChangesCalled  bool
	ChangesChanges *plans.ChangesSync

	StateCalled bool
	StateState  *states.SyncState
}

// MockEvalContext implements EvalContext
var _ EvalContext = (*MockEvalContext)(nil)

func (c *MockEvalContext) Stopped() <-chan struct{} {
	c.StoppedCalled = true
	return c.StoppedValue
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

func (c *MockEvalContext) InitProvider(t string, addr addrs.ProviderConfig) (providers.Interface, error) {
	c.InitProviderCalled = true
	c.InitProviderType = t
	c.InitProviderAddr = addr
	return c.InitProviderProvider, c.InitProviderError
}

func (c *MockEvalContext) Provider(addr addrs.AbsProviderConfig) providers.Interface {
	c.ProviderCalled = true
	c.ProviderAddr = addr
	return c.ProviderProvider
}

func (c *MockEvalContext) ProviderSchema(addr addrs.AbsProviderConfig) *ProviderSchema {
	c.ProviderSchemaCalled = true
	c.ProviderSchemaAddr = addr
	return c.ProviderSchemaSchema
}

func (c *MockEvalContext) CloseProvider(addr addrs.ProviderConfig) error {
	c.CloseProviderCalled = true
	c.CloseProviderAddr = addr
	return nil
}

func (c *MockEvalContext) ConfigureProvider(addr addrs.ProviderConfig, cfg cty.Value) tfdiags.Diagnostics {
	c.ConfigureProviderCalled = true
	c.ConfigureProviderAddr = addr
	c.ConfigureProviderConfig = cfg
	return c.ConfigureProviderDiags
}

func (c *MockEvalContext) ProviderInput(addr addrs.ProviderConfig) map[string]cty.Value {
	c.ProviderInputCalled = true
	c.ProviderInputAddr = addr
	return c.ProviderInputValues
}

func (c *MockEvalContext) SetProviderInput(addr addrs.ProviderConfig, vals map[string]cty.Value) {
	c.SetProviderInputCalled = true
	c.SetProviderInputAddr = addr
	c.SetProviderInputValues = vals
}

func (c *MockEvalContext) InitProvisioner(n string) (ResourceProvisioner, error) {
	c.InitProvisionerCalled = true
	c.InitProvisionerName = n
	return c.InitProvisionerProvisioner, c.InitProvisionerError
}

func (c *MockEvalContext) Provisioner(n string) ResourceProvisioner {
	c.ProvisionerCalled = true
	c.ProvisionerName = n
	return c.ProvisionerProvisioner
}

func (c *MockEvalContext) ProvisionerSchema(n string) *configschema.Block {
	c.ProvisionerSchemaCalled = true
	c.ProvisionerSchemaName = n
	return c.ProvisionerSchemaSchema
}

func (c *MockEvalContext) CloseProvisioner(n string) error {
	c.CloseProvisionerCalled = true
	c.CloseProvisionerName = n
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

func (c *MockEvalContext) EvaluationScope(self addrs.Referenceable, keyData InstanceKeyEvalData) *lang.Scope {
	c.EvaluationScopeCalled = true
	c.EvaluationScopeSelf = self
	c.EvaluationScopeKeyData = keyData
	return c.EvaluationScopeScope
}

func (c *MockEvalContext) Interpolate(
	config *config.RawConfig, resource *Resource) (*ResourceConfig, error) {
	c.InterpolateCalled = true
	c.InterpolateConfig = config
	c.InterpolateResource = resource
	return c.InterpolateConfigResult, c.InterpolateError
}

func (c *MockEvalContext) InterpolateProvider(
	config *config.ProviderConfig, resource *Resource) (*ResourceConfig, error) {
	c.InterpolateProviderCalled = true
	c.InterpolateProviderConfig = config
	c.InterpolateProviderResource = resource
	return c.InterpolateProviderConfigResult, c.InterpolateError
}

func (c *MockEvalContext) Path() addrs.ModuleInstance {
	c.PathCalled = true
	return c.PathPath
}

func (c *MockEvalContext) SetModuleCallArguments(n addrs.ModuleCallInstance, values map[string]cty.Value) {
	c.SetModuleCallArgumentsCalled = true
	c.SetModuleCallArgumentsModule = n
	c.SetModuleCallArgumentsValues = values
}

func (c *MockEvalContext) Changes() *plans.ChangesSync {
	c.ChangesCalled = true
	return c.ChangesChanges
}

func (c *MockEvalContext) State() *states.SyncState {
	c.StateCalled = true
	return c.StateState
}
