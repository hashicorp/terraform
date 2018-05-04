package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/lang"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/config"
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
	InitProviderProvider ResourceProvider
	InitProviderError    error

	ProviderCalled   bool
	ProviderAddr     addrs.AbsProviderConfig
	ProviderProvider ResourceProvider

	ProviderSchemaCalled bool
	ProviderSchemaAddr   addrs.AbsProviderConfig
	ProviderSchemaSchema *ProviderSchema

	CloseProviderCalled   bool
	CloseProviderAddr     addrs.ProviderConfig
	CloseProviderProvider ResourceProvider

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

	EvaluateBlockCalled       bool
	EvaluateBlockBody         hcl.Body
	EvaluateBlockSchema       *configschema.Block
	EvaluateBlockSelf         addrs.Referenceable
	EvaluateBlockKey          addrs.InstanceKey
	EvaluateBlockResult       cty.Value
	EvaluateBlockExpandedBody hcl.Body
	EvaluateBlockDiags        tfdiags.Diagnostics

	EvaluateExprCalled   bool
	EvaluateExprExpr     hcl.Expression
	EvaluateExprWantType cty.Type
	EvaluateExprSelf     addrs.Referenceable
	EvaluateExprResult   cty.Value
	EvaluateExprDiags    tfdiags.Diagnostics

	EvaluationScopeCalled bool
	EvaluationScopeSelf   addrs.Referenceable
	EvaluationScopeKey    addrs.InstanceKey
	EvaluationScopeScope  *lang.Scope

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

	DiffCalled bool
	DiffDiff   *Diff
	DiffLock   *sync.RWMutex

	StateCalled bool
	StateState  *State
	StateLock   *sync.RWMutex
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

func (c *MockEvalContext) InitProvider(t string, addr addrs.ProviderConfig) (ResourceProvider, error) {
	c.InitProviderCalled = true
	c.InitProviderType = t
	c.InitProviderAddr = addr
	return c.InitProviderProvider, c.InitProviderError
}

func (c *MockEvalContext) Provider(addr addrs.AbsProviderConfig) ResourceProvider {
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

func (c *MockEvalContext) EvaluateBlock(body hcl.Body, schema *configschema.Block, self addrs.Referenceable, key addrs.InstanceKey) (cty.Value, hcl.Body, tfdiags.Diagnostics) {
	c.EvaluateBlockCalled = true
	c.EvaluateBlockBody = body
	c.EvaluateBlockSchema = schema
	c.EvaluateBlockSelf = self
	c.EvaluateBlockKey = key
	return c.EvaluateBlockResult, c.EvaluateBlockExpandedBody, c.EvaluateBlockDiags
}

func (c *MockEvalContext) EvaluateExpr(expr hcl.Expression, wantType cty.Type, self addrs.Referenceable) (cty.Value, tfdiags.Diagnostics) {
	c.EvaluateExprCalled = true
	c.EvaluateExprExpr = expr
	c.EvaluateExprWantType = wantType
	c.EvaluateExprSelf = self
	return c.EvaluateExprResult, c.EvaluateExprDiags
}

func (c *MockEvalContext) EvaluationScope(self addrs.Referenceable, key addrs.InstanceKey) *lang.Scope {
	c.EvaluationScopeCalled = true
	c.EvaluationScopeSelf = self
	c.EvaluationScopeKey = key
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

func (c *MockEvalContext) Diff() (*Diff, *sync.RWMutex) {
	c.DiffCalled = true
	return c.DiffDiff, c.DiffLock
}

func (c *MockEvalContext) State() (*State, *sync.RWMutex) {
	c.StateCalled = true
	return c.StateState, c.StateLock
}
