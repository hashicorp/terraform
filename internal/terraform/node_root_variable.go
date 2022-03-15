package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NodeRootVariable represents a root variable input.
type NodeRootVariable struct {
	Addr   addrs.InputVariable
	Config *configs.Variable

	// RawValue is the value for the variable set from outside Terraform
	// Core, such as on the command line, or from an environment variable,
	// or similar. This is the raw value that was provided, not yet
	// converted or validated, and can be nil for a variable that isn't
	// set at all.
	RawValue *InputValue
}

var (
	_ GraphNodeModuleInstance = (*NodeRootVariable)(nil)
	_ GraphNodeReferenceable  = (*NodeRootVariable)(nil)
)

func (n *NodeRootVariable) Name() string {
	return n.Addr.String()
}

// GraphNodeModuleInstance
func (n *NodeRootVariable) Path() addrs.ModuleInstance {
	return addrs.RootModuleInstance
}

func (n *NodeRootVariable) ModulePath() addrs.Module {
	return addrs.RootModule
}

// GraphNodeReferenceable
func (n *NodeRootVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// GraphNodeExecutable
func (n *NodeRootVariable) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	// Root module variables are special in that they are provided directly
	// by the caller (usually, the CLI layer) and so we don't really need to
	// evaluate them in the usual sense, but we do need to process the raw
	// values given by the caller to match what the module is expecting, and
	// make sure the values are valid.
	var diags tfdiags.Diagnostics

	addr := addrs.RootModuleInstance.InputVariable(n.Addr.Name)
	log.Printf("[TRACE] NodeRootVariable: evaluating %s", addr)

	if n.Config == nil {
		// Because we build NodeRootVariable from configuration in the normal
		// case it's strange to get here, but we tolerate it to allow for
		// tests that might not populate the inputs fully.
		return nil
	}

	givenVal := n.RawValue
	if givenVal == nil {
		// We'll use cty.NilVal to represent the variable not being set at
		// all, which for historical reasons is unfortunately different than
		// explicitly setting it to null in some cases. In normal code we
		// should never get here because all variables should have raw
		// values, but we can get here in some historical tests that call
		// in directly and don't necessarily obey the rules.
		givenVal = &InputValue{
			Value:      cty.NilVal,
			SourceType: ValueFromUnknown,
		}
	}

	finalVal, moreDiags := prepareFinalInputVariableValue(
		addr,
		givenVal,
		n.Config,
	)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		// No point in proceeding to validations then, because they'll
		// probably fail trying to work with a value of the wrong type.
		return diags
	}

	ctx.SetRootModuleArgument(addr.Variable, finalVal)

	moreDiags = evalVariableValidations(
		addrs.RootModuleInstance.InputVariable(n.Addr.Name),
		n.Config,
		nil, // not set for root module variables
		ctx,
	)
	diags = diags.Append(moreDiags)
	return diags
}

// dag.GraphNodeDotter impl.
func (n *NodeRootVariable) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}
