package applying

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// resourceActions gathers together all of the action instances for a
// particular resource, associating each with the addresses of objects they
// depend on.
type resourceActions struct {
	Addr              addrs.AbsResource
	SetMeta           *resourceSetMetaAction
	Instances         map[addrs.InstanceKey]*resourceInstanceActions
	Cleanup           *resourceCleanupAction
	ProviderConfigRef addrs.AbsProviderConfig
	Dependencies      []addrs.Referenceable
}

func (ra *resourceActions) AllRequire(a action, g *dag.AcyclicGraph) {
	ra.AllResourceOnlyRequire(a, g)
	ra.AllInstancesRequire(a, g)
}

func (ra *resourceActions) AllRequiredBy(a action, g *dag.AcyclicGraph) {
	ra.AllResourceOnlyRequiredBy(a, g)
	ra.AllInstancesRequiredBy(a, g)
}

func (ra *resourceActions) AllResourceOnlyRequire(a action, g *dag.AcyclicGraph) {
	if ra.SetMeta != nil {
		g.Connect(dag.BasicEdge(ra.SetMeta, a))
	}
	if ra.Cleanup != nil {
		g.Connect(dag.BasicEdge(ra.Cleanup, a))
	}
}

func (ra *resourceActions) AllResourceOnlyRequiredBy(a action, g *dag.AcyclicGraph) {
	if ra.SetMeta != nil {
		g.Connect(dag.BasicEdge(a, ra.SetMeta))
	}
	if ra.Cleanup != nil {
		g.Connect(dag.BasicEdge(a, ra.Cleanup))
	}
}

func (ra *resourceActions) AllInstancesRequire(a action, g *dag.AcyclicGraph) {
	for _, riActions := range ra.Instances {
		riActions.AllRequire(a, g)
	}
}

func (ra *resourceActions) AllInstancesRequiredBy(a action, g *dag.AcyclicGraph) {
	for _, riActions := range ra.Instances {
		riActions.AllRequiredBy(a, g)
	}
}

// resourceInstanceActions gathers together the action instances for a
// particular resource instance and the addresses of objects they depend on.
type resourceInstanceActions struct {
	Addr           addrs.AbsResourceInstance
	CreateUpdate   *resourceInstanceNonDestroyChangeAction
	Destroy        *resourceInstanceDestroyChangeAction
	DestroyDeposed map[states.DeposedKey]*resourceInstanceDestroyChangeAction
}

func (ria *resourceInstanceActions) AllRequire(a action, g *dag.AcyclicGraph) {
	if ria.CreateUpdate != nil {
		g.Connect(dag.BasicEdge(ria.CreateUpdate, a))
	}
	if ria.Destroy != nil {
		g.Connect(dag.BasicEdge(ria.Destroy, a))
	}
	for _, destroyDeposedAction := range ria.DestroyDeposed {
		g.Connect(dag.BasicEdge(destroyDeposedAction, a))
	}
}

func (ria *resourceInstanceActions) AllRequiredBy(a action, g *dag.AcyclicGraph) {
	if ria.CreateUpdate != nil {
		g.Connect(dag.BasicEdge(a, ria.CreateUpdate))
	}
	if ria.Destroy != nil {
		g.Connect(dag.BasicEdge(a, ria.Destroy))
	}
	for _, destroyDeposedAction := range ria.DestroyDeposed {
		g.Connect(dag.BasicEdge(a, destroyDeposedAction))
	}
}

// resourceInstanceNonDestroyChangeAction is an action that handles executing a
// planned change with any action other than plans.Delete.
type resourceInstanceNonDestroyChangeAction struct {
	Addr          addrs.AbsResourceInstance
	Action        plans.Action
	Config        *configs.Resource
	PriorObj      cty.Value
	PlannedNewObj cty.Value
}

func (a *resourceInstanceNonDestroyChangeAction) Name() string {
	return fmt.Sprintf("%s %s", a.Action, a.Addr)
}

func (a *resourceInstanceNonDestroyChangeAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Resource instance change action not yet implemented",
		"The prototype apply codepath does not yet support making resource instance changes.",
	))

	return diags
}

// resourceInstanceDestroyChangeAction is an action that handles executing a
// plans.Delete change for a resource instance.
type resourceInstanceDestroyChangeAction struct {
	Addr       addrs.AbsResourceInstance
	DeposedKey states.DeposedKey
	Action     plans.Action
	PriorObj   cty.Value
}

func (a *resourceInstanceDestroyChangeAction) Name() string {
	return fmt.Sprintf("%s %s", a.Action, a.Addr)
}

func (a *resourceInstanceDestroyChangeAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if a.Action != plans.Delete {
		// Currently this action type is only used for delete.
		panic(fmt.Sprintf("resourceInstanceDestroyChangeAction with non-Delete action %s", a.Action))
	}

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Resource instance change action not yet implemented",
		"The prototype apply codepath does not yet support making resource instance changes.",
	))

	return diags
}

// resourceSetMetaAction is an action that sets metadata that applies
// to a resource itself, rather than to its instances individually.
type resourceSetMetaAction struct {
	Addr           addrs.AbsResource
	ForEach        hcl.Expression
	Count          hcl.Expression
	ProviderConfig addrs.AbsProviderConfig
}

func (a *resourceSetMetaAction) Name() string {
	return fmt.Sprintf("Set metadata for %s", a.Addr)
}

func (a *resourceSetMetaAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Resource set metadata action not yet implemented",
		"The prototype apply codepath does not yet support setting resource instance metadata.",
	))

	return diags
}

// resourceCleanupAction is an action that will delete the empty shell of a
// resource from the state, assuming it actually is empty.
//
// This action should only be generated when the resource configuration has
// been removed altogether. If the resource configuration still exists but
// has count = 0 or for_each = {} then removing the empty resource shell
// would not be appropriate.
type resourceCleanupAction struct {
	Addr addrs.AbsResource
}

func (a *resourceCleanupAction) Name() string {
	return fmt.Sprintf("Delete %s from the state", a.Addr)
}

func (a *resourceCleanupAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Resource cleanup action not yet implemented",
		"The prototype apply codepath does not yet support resource cleanup.",
	))

	return diags
}
