package planner

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	opentracing "github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
	"github.com/zclconf/go-cty/cty"
)

// NOTE: For the moment, in prototyping, this only really supports _managed_
// resource instances, and will do strange stuff if given a data resource
// instance.

type resourceInstance struct {
	planner *planner
	addr    addrs.AbsResourceInstance
}

func (ri resourceInstance) Addr() addrs.AbsResourceInstance {
	return ri.addr
}

func (ri resourceInstance) InstanceKey() addrs.InstanceKey {
	return ri.addr.Resource.Key
}

func (ri resourceInstance) Resource() resource {
	return resource{
		planner: ri.planner,
		addr:    ri.addr.ContainingResource(),
	}
}

func (ri resourceInstance) ModuleInstance() moduleInstance {
	return moduleInstance{
		planner: ri.planner,
		addr:    ri.addr.Module,
	}
}

func (ri resourceInstance) IsTargeted() bool {
	targetAddrs := ri.planner.TargetAddrs()
	if len(targetAddrs) == 0 {
		return true // everything is included by default
	}
	for _, targetAddr := range targetAddrs {
		if targetAddr.TargetContains(ri.addr) {
			return true
		}
	}
	return false
}

func (ri resourceInstance) RepetitionData(ctx context.Context) instances.RepetitionData {
	return repetitionDataForInstance(ctx, ri.Addr().Resource.Key, ri.Resource().EachValueForInstance)
}

func (ri resourceInstance) PrevRunValue(ctx context.Context) cty.Value {
	is := ri.PrevRunState(ctx)
	if is == nil {
		schema, _ := ri.Resource().InConfig().Schema(ctx)
		return cty.NullVal(schema.ImpliedType())
	}
	return is.Value
}

func (ri resourceInstance) PrevRunState(ctx context.Context) *states.ResourceInstanceObject {
	retI := ri.planner.DataRequest(ctx, resourceInstancePrevRunStateRequest{ri})
	if retI == nil {
		return nil
	}
	return retI.(*states.ResourceInstanceObject)
}

func (ri resourceInstance) CurrentValue(ctx context.Context) cty.Value {
	is := ri.CurrentState(ctx)
	if is == nil {
		schema, _ := ri.Resource().InConfig().Schema(ctx)
		return cty.NullVal(schema.ImpliedType())
	}
	return is.Value
}

func (ri resourceInstance) CurrentState(ctx context.Context) *states.ResourceInstanceObject {
	retI := ri.planner.DataRequest(ctx, resourceInstanceCurrentStateRequest{ri})
	if retI == nil {
		return nil
	}
	return retI.(*states.ResourceInstanceObject)
}

func (ri resourceInstance) PlannedNewValue(ctx context.Context) cty.Value {
	change := ri.PlannedChange(ctx)
	if change == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	return change.After
}

func (ri resourceInstance) PlannedChange(ctx context.Context) *plans.ResourceInstanceChange {
	ret := ri.planner.DataRequest(ctx, resourceInstancePlannedChangeRequest{ri})
	if ret == nil {
		return nil
	}
	return ret.(*plans.ResourceInstanceChange)
}

type resourceInstancePrevRunStateRequest struct {
	inst resourceInstance
}

type resourceInstancePrevRunStateRequestKey struct {
	instKey addrs.UniqueKey
}

func (req resourceInstancePrevRunStateRequest) requestKey() interface{} {
	return resourceInstancePrevRunStateRequestKey{req.inst.Addr().UniqueKey()}
}

func (req resourceInstancePrevRunStateRequest) handleDataRequest(ctx context.Context, p *planner) interface{} {
	inst := req.inst

	span, _ := opentracing.StartSpanFromContext(ctx, "resourceInstance.PrevRunValue")
	span.LogFields(
		tracelog.String("resourceInstance", inst.Addr().String()),
	)
	defer span.Finish()

	is := inst.planner.PrevRunState().ResourceInstance(inst.Addr())
	if is == nil || is.Current == nil {
		return nil
	}

	objSrc := is.Current

	// We will give the provider an opportunity to upgrade the
	// previously-stored value, in case it was created by an earlier version
	// with a different schema.
	provider, err := inst.Resource().InConfig().ProviderConfig().Instance()
	if err != nil {
		p.AddDiagnostics(err)
		return cty.DynamicVal
	}
	defer provider.Close()

	resp := provider.UpgradeResourceState(providers.UpgradeResourceStateRequest{
		TypeName:        inst.Addr().Resource.Resource.Type,
		Version:         int64(objSrc.SchemaVersion),
		RawStateJSON:    objSrc.AttrsJSON,
		RawStateFlatmap: objSrc.AttrsFlat,
	})
	p.AddDiagnostics(resp.Diagnostics)
	retVal := resp.UpgradedState
	if resp.Diagnostics.HasErrors() {
		retVal = cty.DynamicVal
	}
	return &states.ResourceInstanceObject{
		Value:               retVal,
		Private:             objSrc.Private,
		Status:              objSrc.Status,
		Dependencies:        objSrc.Dependencies,
		CreateBeforeDestroy: objSrc.CreateBeforeDestroy,
	}
}

type resourceInstanceCurrentStateRequest struct {
	inst resourceInstance
}

type resourceInstanceCurrentStateRequestKey struct {
	instKey addrs.UniqueKey
}

func (req resourceInstanceCurrentStateRequest) requestKey() interface{} {
	return resourceInstanceCurrentStateRequestKey{req.inst.Addr().UniqueKey()}
}

func (req resourceInstanceCurrentStateRequest) handleDataRequest(ctx context.Context, p *planner) interface{} {
	inst := req.inst

	span, _ := opentracing.StartSpanFromContext(ctx, "resourceInstance.CurrentValue")
	span.LogFields(
		tracelog.String("resourceInstance", inst.Addr().String()),
	)
	defer span.Finish()

	// The "current value" is determined by the provider's "ReadResource"
	// function, which takes the previous run value as its input so that
	// it can see any identifiers it needs to find the corresponding
	// remote object.
	prevRunState := inst.PrevRunState(ctx)
	if prevRunState == nil {
		return nil
	}
	if !prevRunState.Value.IsWhollyKnown() {
		// This suggests an error upstream, because previous run values can
		// never really be unknown. We'll just pass on the unknown-ness
		// to let the call stack unwind.
		return &states.ResourceInstanceObject{
			Value:               cty.DynamicVal,
			Private:             prevRunState.Private,
			Status:              prevRunState.Status,
			Dependencies:        prevRunState.Dependencies,
			CreateBeforeDestroy: prevRunState.CreateBeforeDestroy,
		}
	}
	if prevRunState.Value.IsNull() {
		// This suggests that we've been asked about a resource instance
		// that's only just been added to the configuration and thus has
		// no previous run state. It therefore has no "current value" either,
		// and thus we should plan to create it.
		return prevRunState
	}

	provider, err := inst.Resource().InConfig().ProviderConfig().Instance()
	if err != nil {
		p.AddDiagnostics(err)
		return cty.DynamicVal
	}
	defer provider.Close()

	resp := provider.ReadResource(providers.ReadResourceRequest{
		TypeName:   inst.Addr().Resource.Resource.Type,
		PriorState: prevRunState.Value,
		// TODO: Also need to preserve "Private" somehow. Maybe we need
		// to pass states.ResourceInstanceObject values between these
		// methods rather than just cty.Value.

		// TODO: Also need to handle ProviderMeta.
		ProviderMeta: cty.NullVal(cty.DynamicPseudoType),
	})
	p.AddDiagnostics(resp.Diagnostics)
	retVal := resp.NewState
	if resp.Diagnostics.HasErrors() {
		retVal = cty.DynamicVal
	}
	return &states.ResourceInstanceObject{
		Value:               retVal,
		Private:             prevRunState.Private,
		Status:              prevRunState.Status,
		Dependencies:        prevRunState.Dependencies,
		CreateBeforeDestroy: prevRunState.CreateBeforeDestroy,
	}
}

type resourceInstancePlannedChangeRequest struct {
	inst resourceInstance
}

func (req resourceInstancePlannedChangeRequest) requestKey() interface{} {
	return resourceInstancePlannedChangeRequestKey{req.inst.Addr().UniqueKey()}
}

func (req resourceInstancePlannedChangeRequest) handleDataRequest(ctx context.Context, p *planner) interface{} {
	inst := req.inst

	span, _ := opentracing.StartSpanFromContext(ctx, "resourceInstance.PlannedChange")
	span.LogFields(
		tracelog.String("resourceInstance", inst.Addr().String()),
	)
	defer span.Finish()

	schema, _ := inst.Resource().InConfig().Schema(ctx)
	provider, err := inst.Resource().InConfig().ProviderConfig().Instance()
	if err != nil {
		p.AddDiagnostics(err)
		return nil
	}
	defer provider.Close()

	action := plans.NoOp
	var before, after cty.Value
	var requiresReplace []cty.Path

	currentState := inst.CurrentState(ctx)
	if currentState != nil {
		before = currentState.Value
	} else {
		before = cty.NullVal(schema.ImpliedType())
	}

	config := inst.Resource().InConfig().Config()
	switch {
	case config != nil:
		repData := inst.RepetitionData(ctx)
		scope := p.ChildInstanceInModuleInstanceExprScope(inst.ModuleInstance().Addr(), repData, nil)
		configVal, diags := scope.EvalBlock(config.Config, schema)
		p.AddDiagnostics(diags)
		if diags.HasErrors() {
			return nil
		}

		proposedNewVal := objchange.ProposedNew(schema, before, configVal)

		var priorPrivate []byte
		if currentState != nil {
			priorPrivate = currentState.Private
		}

		resp := provider.PlanResourceChange(providers.PlanResourceChangeRequest{
			TypeName:         inst.Addr().Resource.Resource.Type,
			PriorState:       before,
			ProposedNewState: proposedNewVal,
			Config:           configVal,
			PriorPrivate:     priorPrivate,

			// TODO: ProviderMeta
			ProviderMeta: cty.NullVal(cty.DynamicPseudoType),
		})
		p.AddDiagnostics(resp.Diagnostics)
		if resp.Diagnostics.HasErrors() {
			return nil
		}

		after = resp.PlannedState
		requiresReplace = resp.RequiresReplace

		errs := objchange.AssertPlanValid(schema, before, configVal, after)
		if len(errs) != 0 {
			for _, err := range errs {
				// TODO: Format these as proper diagnostics
				p.AddDiagnostics(err)
			}
			return nil
		}

		action = objchange.ActionForChange(before, after)
		if action == plans.Update && len(requiresReplace) != 0 {
			if inst.Resource().InConfig().Config().Managed.CreateBeforeDestroy {
				action = plans.CreateThenDelete
			} else {
				action = plans.DeleteThenCreate
			}
		}

	default:
		// If there's no configuration but there's a current value then
		// we always plan to delete, without any input from the provider.
		if !before.IsNull() {
			action = plans.Delete
		}
		after = cty.NullVal(schema.ImpliedType())
	}

	var requiresReplaceSet cty.PathSet
	if len(requiresReplace) != 0 {
		requiresReplaceSet = cty.NewPathSet(requiresReplace...)
	}

	change := &plans.ResourceInstanceChange{
		Addr: inst.Addr(),
		Change: plans.Change{
			Action: action,
			Before: before,
			After:  after,
		},
		ProviderAddr:    inst.Resource().InConfig().ProviderConfig().Addr(),
		RequiredReplace: requiresReplaceSet,
	}
	return change
}

type resourceInstancePlannedChangeRequestKey struct {
	instKey addrs.UniqueKey
}
