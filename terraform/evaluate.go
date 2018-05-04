package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/lang"
)

// Evaluator provides the necessary contextual data for evaluating expressions
// for a particular walk operation.
type Evaluator struct {
	// Operation defines what type of operation this evaluator is being used
	// for.
	Operation walkOperation

	// Meta is contextual metadata about the current operation.
	Meta *ContextMeta

	// Config is the root node in the configuration tree.
	Config *configs.Config

	// RootVariableValues is a map of values for variables defined in the
	// root module, passed in from external sources. This must not be
	// modified during evaluation.
	RootVariableValues map[string]*InputValue

	// State is the current state. During some operations this structure
	// is mutated concurrently, and so it must be accessed only while holding
	// StateLock.
	State     *State
	StateLock *sync.RWMutex
}

// Scope creates an evaluation scope for the given module path and optional
// resource.
//
// If the "self" argument is nil then the "self" object is not available
// in evaluated expressions. Otherwise, it behaves as an alias for the given
// address.
func (e *Evaluator) Scope(data lang.Data, self addrs.Referenceable) *lang.Scope {
	return &lang.Scope{
		Data:     data,
		SelfAddr: self,
		PureOnly: e.Operation != walkApply && e.Operation != walkDestroy,
		BaseDir:  ".", // Always current working directory for now.
	}
}

// evaluationStateData is an implementation of lang.Data that resolves
// references primarily (but not exclusively) using information from a State.
type evaluationStateData struct {
	Evaluator *Evaluator

	// ModulePath is the path through the dynamic module tree to the module
	// that references will be resolved relative to.
	ModulePath addrs.ModuleInstance
}

// evaluationStateData must implement lang.Data
var _ lang.Data = (*evaluationStateData)(nil)

func (d *evaluationStateData) GetCountAttr(addrs.CountAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	panic("not yet implemented")
}

func (d *evaluationStateData) GetInputVariable(addrs.InputVariable, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	panic("not yet implemented")
}

func (d *evaluationStateData) GetLocalValue(addrs.LocalValue, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	panic("not yet implemented")
}

func (d *evaluationStateData) GetModuleInstance(addrs.ModuleCallInstance, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	panic("not yet implemented")
}

func (d *evaluationStateData) GetModuleInstanceOutput(addrs.ModuleCallOutput, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	panic("not yet implemented")
}

func (d *evaluationStateData) GetPathAttr(addrs.PathAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	panic("not yet implemented")
}

func (d *evaluationStateData) GetResourceInstance(addrs.ResourceInstance, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	panic("not yet implemented")
}

func (d *evaluationStateData) GetTerraformAttr(addrs.TerraformAttr, tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	panic("not yet implemented")
}
