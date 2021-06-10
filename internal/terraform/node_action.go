package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeExpandPlannableMovedAction struct {
	Module addrs.Module
	Config *configs.MovedAction
}

var (
	_ dag.NamedVertex            = (*nodeExpandPlannableMovedAction)(nil)
	_ GraphNodeModulePath        = (*nodeExpandPlannableMovedAction)(nil)
	_ GraphNodeDynamicExpandable = (*nodeExpandPlannableMovedAction)(nil)
)

// NamedVertex
func (n *nodeExpandPlannableMovedAction) Name() string {
	return fmt.Sprintf("moved[%s->%s]", n.Config.From.Subject, n.Config.To.Subject)
}

// GraphNodeModulePath
func (n *nodeExpandPlannableMovedAction) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeDynamicExpandable
func (n *nodeExpandPlannableMovedAction) DynamicExpand(ctx EvalContext) (*Graph, error) {
	expander := ctx.InstanceExpander()

	var g Graph
	for _, module := range expander.ExpandModule(n.Module) {
		o := &NodeMovedAction{
			Module: module,
			Config: n.Config,
		}
		g.Add(o)
	}
	return &g, nil
}

type NodeMovedAction struct {
	Module addrs.ModuleInstance
	Config *configs.MovedAction
}

var (
	_ dag.NamedVertex     = (*NodeMovedAction)(nil)
	_ GraphNodeModulePath = (*NodeMovedAction)(nil)
	_ GraphNodeExecutable = (*NodeMovedAction)(nil)
)

// NamedVertex
func (n *NodeMovedAction) Name() string {
	return fmt.Sprintf("moved[%s->%s]", n.Config.From.Subject, n.Config.To.Subject)
}

// GraphNodeModulePath
func (n *NodeMovedAction) ModulePath() addrs.Module {
	return n.Module.Module()
}

// GraphNodeExecutable
func (n *NodeMovedAction) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	state := ctx.State()
	if state == nil {
		return
	}

	var fromResource addrs.AbsResource

	switch fr := n.Config.From.Subject.(type) {
	case addrs.AbsResource:
		fromResource = fr
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid from address",
			Detail: fmt.Sprintf(
				"From must be a resource, but is %q",
				n.Config.From.Subject.String(),
			),
			Subject: n.Config.DeclRange.Ptr(),
		})
	}

	var toResource addrs.AbsResource

	switch tr := n.Config.To.Subject.(type) {
	case addrs.AbsResource:
		toResource = tr
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid to address",
			Detail: fmt.Sprintf(
				"To must be a resource, but is %q",
				n.Config.To.Subject.String(),
			),
			Subject: n.Config.DeclRange.Ptr(),
		})
	}

	if diags.HasErrors() {
		return diags
	}

	// FIXME this feels wrong
	if op == walkValidate {
		return
	}

	rs := state.Resource(fromResource)
	if rs == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No resource state found",
			fmt.Sprintf("There is no existing state for %v", fromResource.String()),
		))
		return diags
	}
	is := rs.Instance(addrs.NoKey)
	if is == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No resource instance state found",
			fmt.Sprintf("There is no existing instance state for %v", fromResource.Instance(addrs.NoKey).String()),
		))
		return diags
	}

	log.Printf("[DEBUG] Moving resource state from %s to %s", fromResource, toResource)
	ctx.Changes().AppendMovedAction(fromResource, toResource)
	state.SetResourceInstanceCurrent(toResource.Instance(addrs.NoKey), is.Current, rs.ProviderConfig)
	state.SetResourceInstanceCurrent(fromResource.Instance(addrs.NoKey), nil, rs.ProviderConfig)

	return diags
}
