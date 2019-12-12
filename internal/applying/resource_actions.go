package applying

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
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
	Instances         map[addrs.InstanceKey]resourceInstanceActions
	ProviderConfigRef addrs.AbsProviderConfig
	Dependencies      []addrs.Referenceable
}

// resourceInstanceActions gathers together the action instances for a
// particular resource instance and the addresses of objects they depend on.
type resourceInstanceActions struct {
	Addr           addrs.AbsResourceInstance
	CreateUpdate   *resourceInstanceChangeAction
	Destroy        *resourceInstanceChangeAction
	DestroyDeposed map[states.DeposedKey]*resourceInstanceChangeAction
}

// resourceInstanceAction is an action that handles executing a planned
// change to a specific resource instance.
type resourceInstanceChangeAction struct {
	Addr          addrs.AbsResourceInstance
	Action        plans.Action
	Config        *configs.Resource
	PriorObj      *cty.Value
	PlannedNewObj *cty.Value
}

func (a *resourceInstanceChangeAction) Name() string {
	return fmt.Sprintf("%s change for %s", a.Action, a.Addr)
}

func (a *resourceInstanceChangeAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

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
	EachMode       states.EachMode
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
