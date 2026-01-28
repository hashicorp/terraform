// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform-svchost/disco"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackmigrate"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type stacksServer struct {
	stacks.UnimplementedStacksServer

	stopper            *stopper
	services           *disco.Disco
	handles            *handleTable
	experimentsAllowed bool

	// providerCacheOverride is a map of provider names to provider factories
	// that should be used instead of the default provider cache. This is used
	// within tests to side load providers without needing a real provider
	// cache.
	providerCacheOverride map[addrs.Provider]providers.Factory
	// providerDependencyLockOverride is an in-memory override of the provider
	// lockfile used for testing when the real provider is side-loaded.
	providerDependencyLockOverride *depsfile.Locks
	// planTimestampOverride is an in-memory override of the plan timestamp used
	// for testing. This just ensures our tests aren't flaky as we can use a
	// constant timestamp for the plan.
	planTimestampOverride *time.Time
}

var (
	_ stacks.StacksServer = (*stacksServer)(nil)

	WorkspaceNameEnvVar = "TF_WORKSPACE"
)

func newStacksServer(stopper *stopper, handles *handleTable, services *disco.Disco, opts *serviceOpts) *stacksServer {
	return &stacksServer{
		stopper:            stopper,
		services:           services,
		handles:            handles,
		experimentsAllowed: opts.experimentsAllowed,
	}
}

func (s *stacksServer) OpenStackConfiguration(ctx context.Context, req *stacks.OpenStackConfiguration_Request) (*stacks.OpenStackConfiguration_Response, error) {
	sourcesHnd := handle[*sourcebundle.Bundle](req.SourceBundleHandle)
	sources := s.handles.SourceBundle(sourcesHnd)
	if sources == nil {
		return nil, status.Error(codes.InvalidArgument, "the given source bundle handle is invalid")
	}

	sourceAddr, err := resolveFinalSourceAddr(req.SourceAddress, sources)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source address: %s", err)
	}

	config, diags := stackconfig.LoadConfigDir(sourceAddr, sources)
	if diags.HasErrors() {
		// For errors in the configuration itself we treat that as a successful
		// result from OpenStackConfiguration but with diagnostics in the
		// response and no source handle.
		return &stacks.OpenStackConfiguration_Response{
			Diagnostics: diagnosticsToProto(diags),
		}, nil
	}

	configHnd, err := s.handles.NewStackConfig(config, sourcesHnd)
	if err != nil {
		// The only reasonable way we can fail here is if the caller made
		// a concurrent call to Dependencies.CloseSourceBundle after we
		// checked the handle validity above. That'd be a very strange thing
		// to do, but in the event it happens we'll just discard the config
		// we loaded (since its source files on disk might be gone imminently)
		// and return an error.
		return nil, status.Errorf(codes.Unknown, "allocating config handle: %s", err)
	}

	// If we get here then we're guaranteed that the source bundle handle
	// cannot be closed until the config handle is closed -- enforced by
	// [handleTable]'s dependency tracking -- and so we can return the config
	// handle. (The caller is required to ensure that the source bundle files
	// on disk are not modified for as long as the source bundle handle remains
	// open, and its lifetime will necessarily exceed the config handle.)
	return &stacks.OpenStackConfiguration_Response{
		StackConfigHandle: configHnd.ForProtobuf(),
		Diagnostics:       diagnosticsToProto(diags),
	}, nil
}

func (s *stacksServer) CloseStackConfiguration(ctx context.Context, req *stacks.CloseStackConfiguration_Request) (*stacks.CloseStackConfiguration_Response, error) {
	hnd := handle[*stackconfig.Config](req.StackConfigHandle)
	err := s.handles.CloseStackConfig(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &stacks.CloseStackConfiguration_Response{}, nil
}

func (s *stacksServer) ValidateStackConfiguration(ctx context.Context, req *stacks.ValidateStackConfiguration_Request) (*stacks.ValidateStackConfiguration_Response, error) {
	cfgHnd := handle[*stackconfig.Config](req.StackConfigHandle)
	cfg := s.handles.StackConfig(cfgHnd)
	if cfg == nil {
		return nil, status.Error(codes.InvalidArgument, "the given stack configuration handle is invalid")
	}
	depsHnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	var deps *depsfile.Locks
	if !depsHnd.IsNil() {
		deps = s.handles.DependencyLocks(depsHnd)
		if deps == nil {
			return nil, status.Error(codes.InvalidArgument, "the given dependency locks handle is invalid")
		}
	} else {
		deps = depsfile.NewLocks()
	}
	providerCacheHnd := handle[*providercache.Dir](req.ProviderCacheHandle)
	var providerCache *providercache.Dir
	if !providerCacheHnd.IsNil() {
		providerCache = s.handles.ProviderPluginCache(providerCacheHnd)
		if providerCache == nil {
			return nil, status.Error(codes.InvalidArgument, "the given provider cache handle is invalid")
		}
	}

	// (providerFactoriesForLocks explicitly supports a nil providerCache)
	providerFactories, err := providerFactoriesForLocks(deps, providerCache)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "provider dependencies are inconsistent: %s", err)
	}

	diags := stackruntime.Validate(ctx, &stackruntime.ValidateRequest{
		Config:             cfg,
		ExperimentsAllowed: s.experimentsAllowed,
		ProviderFactories:  providerFactories,
		DependencyLocks:    *deps,
	})
	return &stacks.ValidateStackConfiguration_Response{
		Diagnostics: diagnosticsToProto(diags),
	}, nil
}

func (s *stacksServer) FindStackConfigurationComponents(ctx context.Context, req *stacks.FindStackConfigurationComponents_Request) (*stacks.FindStackConfigurationComponents_Response, error) {
	cfgHnd := handle[*stackconfig.Config](req.StackConfigHandle)
	cfg := s.handles.StackConfig(cfgHnd)
	if cfg == nil {
		return nil, status.Error(codes.InvalidArgument, "the given stack configuration handle is invalid")
	}

	return &stacks.FindStackConfigurationComponents_Response{
		Config: stackConfigMetaforProto(cfg.Root, stackaddrs.RootStack),
	}, nil
}

func stackConfigMetaforProto(cfgNode *stackconfig.ConfigNode, stackAddr stackaddrs.Stack) *stacks.FindStackConfigurationComponents_StackConfig {
	ret := &stacks.FindStackConfigurationComponents_StackConfig{
		Components:     make(map[string]*stacks.FindStackConfigurationComponents_Component),
		EmbeddedStacks: make(map[string]*stacks.FindStackConfigurationComponents_EmbeddedStack),
		InputVariables: make(map[string]*stacks.FindStackConfigurationComponents_InputVariable),
		OutputValues:   make(map[string]*stacks.FindStackConfigurationComponents_OutputValue),
		Removed:        make(map[string]*stacks.FindStackConfigurationComponents_Removed),
	}

	for name, cc := range cfgNode.Stack.Components {
		cProto := &stacks.FindStackConfigurationComponents_Component{
			SourceAddr:    cc.FinalSourceAddr.String(),
			ComponentAddr: stackaddrs.Config(stackAddr, stackaddrs.Component{Name: cc.Name}).String(),
		}
		switch {
		case cc.ForEach != nil:
			cProto.Instances = stacks.FindStackConfigurationComponents_FOR_EACH
		default:
			cProto.Instances = stacks.FindStackConfigurationComponents_SINGLE
		}
		ret.Components[name] = cProto
	}

	for name, sn := range cfgNode.Children {
		sc := cfgNode.Stack.EmbeddedStacks[name]
		sProto := &stacks.FindStackConfigurationComponents_EmbeddedStack{
			SourceAddr: sn.Stack.SourceAddr.String(),
			Config:     stackConfigMetaforProto(sn, stackAddr.Child(name)),
		}
		switch {
		case sc.ForEach != nil:
			sProto.Instances = stacks.FindStackConfigurationComponents_FOR_EACH
		default:
			sProto.Instances = stacks.FindStackConfigurationComponents_SINGLE
		}
		ret.EmbeddedStacks[name] = sProto
	}

	for name, vc := range cfgNode.Stack.InputVariables {
		vProto := &stacks.FindStackConfigurationComponents_InputVariable{
			Optional:  !vc.DefaultValue.IsNull(),
			Sensitive: vc.Sensitive,
			Ephemeral: vc.Ephemeral,
		}
		ret.InputVariables[name] = vProto
	}

	for name, oc := range cfgNode.Stack.OutputValues {
		oProto := &stacks.FindStackConfigurationComponents_OutputValue{
			Sensitive: oc.Sensitive,
			Ephemeral: oc.Ephemeral,
		}
		ret.OutputValues[name] = oProto
	}

	// Currently Components are the only thing that can be removed
	for name, rc := range cfgNode.Stack.RemovedComponents.All() {
		var blocks []*stacks.FindStackConfigurationComponents_Removed_Block
		for _, rc := range rc {
			relativeAddress := rc.From.TargetConfigComponent()
			cProto := &stacks.FindStackConfigurationComponents_Removed_Block{
				SourceAddr:    rc.FinalSourceAddr.String(),
				ComponentAddr: stackaddrs.Config(append(stackAddr, relativeAddress.Stack...), relativeAddress.Item).String(),
				Destroy:       rc.Destroy,
			}
			switch {
			case rc.ForEach != nil:
				cProto.Instances = stacks.FindStackConfigurationComponents_FOR_EACH
			default:
				cProto.Instances = stacks.FindStackConfigurationComponents_SINGLE
			}
			blocks = append(blocks, cProto)
		}
		relativeAddress := rc[0].From.TargetConfigComponent()
		ret.Removed[name.String()] = &stacks.FindStackConfigurationComponents_Removed{
			// in order to ensure as much backwards and forwards compatibility
			// as possible, we're going to set the deprecated single fields
			// with the first run block

			SourceAddr: rc[0].FinalSourceAddr.String(),
			Instances: func() stacks.FindStackConfigurationComponents_Instances {
				switch {
				case rc[0].ForEach != nil:
					return stacks.FindStackConfigurationComponents_FOR_EACH
				default:
					return stacks.FindStackConfigurationComponents_SINGLE
				}
			}(),
			ComponentAddr: stackaddrs.Config(append(stackAddr, relativeAddress.Stack...), relativeAddress.Item).String(),
			Destroy:       rc[0].Destroy,

			// We return all the values here:

			Blocks: blocks,
		}
	}

	return ret
}

func (s *stacksServer) OpenState(stream stacks.Stacks_OpenStateServer) error {
	loader := stackstate.NewLoader()
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			break // All done!
		} else if err != nil {
			return err
		}
		err = loader.AddRaw(item.Raw.Key, item.Raw.Value)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid raw state element: %s", err)
		}
	}

	hnd := s.handles.NewStackState(loader.State())
	return stream.SendAndClose(&stacks.OpenStackState_Response{
		StateHandle: hnd.ForProtobuf(),
	})
}

func (s *stacksServer) CloseState(ctx context.Context, req *stacks.CloseStackState_Request) (*stacks.CloseStackState_Response, error) {
	hnd := handle[*stackstate.State](req.StateHandle)
	err := s.handles.CloseStackState(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &stacks.CloseStackState_Response{}, nil
}

func (s *stacksServer) PlanStackChanges(req *stacks.PlanStackChanges_Request, evts stacks.Stacks_PlanStackChangesServer) error {
	ctx := evts.Context()
	syncEvts := newSyncStreamingRPCSender(evts)
	evts = nil // Prevent accidental unsynchronized usage of this server

	cfgHnd := handle[*stackconfig.Config](req.StackConfigHandle)
	cfg := s.handles.StackConfig(cfgHnd)
	if cfg == nil {
		return status.Error(codes.InvalidArgument, "the given stack configuration handle is invalid")
	}
	depsHnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	var deps *depsfile.Locks
	if s.providerDependencyLockOverride != nil {
		deps = s.providerDependencyLockOverride
	} else if !depsHnd.IsNil() {
		deps = s.handles.DependencyLocks(depsHnd)
		if deps == nil {
			return status.Error(codes.InvalidArgument, "the given dependency locks handle is invalid")
		}
	} else {
		deps = depsfile.NewLocks()
	}
	providerCacheHnd := handle[*providercache.Dir](req.ProviderCacheHandle)
	var providerCache *providercache.Dir
	if !providerCacheHnd.IsNil() {
		providerCache = s.handles.ProviderPluginCache(providerCacheHnd)
		if providerCache == nil {
			return status.Error(codes.InvalidArgument, "the given provider cache handle is invalid")
		}
	}
	// NOTE: providerCache can be nil if no handle was provided, in which
	// case the call can only use built-in providers. All code below
	// must avoid panicking when providerCache is nil, but is allowed to
	// return an InvalidArgument error in that case.

	if req.PreviousStateHandle != 0 && len(req.PreviousState) != 0 {
		return status.Error(codes.InvalidArgument, "must not set both previous_state_handle and previous_state")
	}

	inputValues, err := externalInputValuesFromProto(req.InputValues)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid input values: %s", err)
	}

	var providerFactories map[addrs.Provider]providers.Factory
	if s.providerCacheOverride != nil {
		// This is only used in tests to side load providers without needing a
		// real provider cache.
		providerFactories = s.providerCacheOverride
	} else {
		// (providerFactoriesForLocks explicitly supports a nil providerCache)
		providerFactories, err = providerFactoriesForLocks(deps, providerCache)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "provider dependencies are inconsistent: %s", err)
		}
	}

	// We'll hook some internal events in the planning process both to generate
	// tracing information if we're in an OpenTelemetry-aware context and
	// to propagate a subset of the events to our client.
	hooks := stackPlanHooks(syncEvts, cfg.Root.Stack.SourceAddr)
	ctx = stackruntime.ContextWithHooks(ctx, hooks)

	var planMode plans.Mode
	switch req.PlanMode {
	case stacks.PlanMode_NORMAL:
		planMode = plans.NormalMode
	case stacks.PlanMode_REFRESH_ONLY:
		planMode = plans.RefreshOnlyMode
	case stacks.PlanMode_DESTROY:
		planMode = plans.DestroyMode
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported planning mode %d", req.PlanMode)
	}

	var prevState *stackstate.State
	if req.PreviousStateHandle != 0 {
		stateHnd := handle[*stackstate.State](req.PreviousStateHandle)
		prevState = s.handles.StackState(stateHnd)
		if prevState == nil {
			return status.Error(codes.InvalidArgument, "the given previous state handle is invalid")
		}
	} else {
		// Deprecated: The previous state is provided inline as a map.
		// FIXME: Remove this old field once our existing clients are updated.
		prevState, err = stackstate.LoadFromProto(req.PreviousState)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "can't load previous state: %s", err)
		}
	}

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	rtReq := stackruntime.PlanRequest{
		PlanMode:           planMode,
		Config:             cfg,
		PrevState:          prevState,
		ProviderFactories:  providerFactories,
		InputValues:        inputValues,
		ExperimentsAllowed: s.experimentsAllowed,
		DependencyLocks:    *deps,

		// planTimestampOverride will be null if not set, so it's fine for
		// us to just set this all the time. In practice, this will only have
		// a value in tests.
		ForcePlanTimestamp: s.planTimestampOverride,
	}
	rtResp := stackruntime.PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	// As a long-running operation, the plan RPC must be able to be stopped. We
	// do this by requesting a stop channel from the stopper, and using it to
	// cancel the planning process.
	stopCh := s.stopper.add()
	defer s.stopper.remove(stopCh)

	// We create a new cancellable context for the stack plan operation to
	// allow us to respond to stop requests.
	planCtx, cancelPlan := context.WithCancel(ctx)
	defer cancelPlan()

	// The actual plan operation runs in the background, and emits events
	// to us via the channels in rtResp before finally closing changesCh
	// to signal that the process is complete.
	go stackruntime.Plan(planCtx, &rtReq, &rtResp)

	emitDiag := func(diag tfdiags.Diagnostic) {
		diags := tfdiags.Diagnostics{diag}
		protoDiags := diagnosticsToProto(diags)
		for _, protoDiag := range protoDiags {
			syncEvts.Send(&stacks.PlanStackChanges_Event{
				Event: &stacks.PlanStackChanges_Event_Diagnostic{
					Diagnostic: protoDiag,
				},
			})
		}
	}

	// There is no strong ordering between the planned changes and the
	// diagnostics, so we need to be prepared for them to arrive in any
	// order. However, stackruntime.Plan does guarantee that it will
	// close changesCh only after it's finished writing to and closing
	// everything else, and so we can assume that once changesCh is
	// closed we only need to worry about whatever's left in the
	// diagsCh buffer.
Events:
	for {
		select {

		case change, ok := <-changesCh:
			if !ok {
				if diagsCh != nil {
					// Async work is done! We do still need to consume the rest
					// of diagsCh before we stop, though, because there might
					// be some extras in the channel's buffer that we didn't
					// get to yet.
					for diag := range diagsCh {
						emitDiag(diag)
					}
				}
				break Events
			}

			protoChange, err := change.PlannedChangeProto()
			if err != nil {
				// Should not get here: it always indicates a bug in
				// PlannedChangeProto or in the code which constructed
				// the change over in package stackeval.
				emitDiag(tfdiags.Sourceless(
					tfdiags.Error,
					"Incorrectly-constructed change",
					fmt.Sprintf(
						"Failed to serialize a %T value for recording in the saved plan: %s.\n\nThis is a bug in Terraform; please report it!",
						protoChange, err,
					),
				))
				continue
			}

			syncEvts.Send(&stacks.PlanStackChanges_Event{
				Event: &stacks.PlanStackChanges_Event_PlannedChange{
					PlannedChange: protoChange,
				},
			})

		case diag, ok := <-diagsCh:
			if !ok {
				// The diagnostics channel has closed, so we'll just stop
				// trying to read from it and wait for changesCh to close,
				// which will be our final signal that everything is done.
				diagsCh = nil
				continue
			}
			emitDiag(diag)

		case <-stopCh:
			// If our stop channel is signalled, we need to cancel the plan.
			// This may result in remaining changes or diagnostics being
			// emitted, so we continue to monitor those channels if they're
			// still active.
			cancelPlan()
		}
	}

	return nil
}

func (s *stacksServer) OpenPlan(stream stacks.Stacks_OpenPlanServer) error {
	loader := stackplan.NewLoader()
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			break // All done!
		} else if err != nil {
			return err
		}
		err = loader.AddRaw(item.Raw)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid raw plan element: %s", err)
		}
	}

	plan, err := loader.Plan()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid raw plan: %s", err)
	}
	hnd := s.handles.NewStackPlan(plan)
	return stream.SendAndClose(&stacks.OpenStackPlan_Response{
		PlanHandle: hnd.ForProtobuf(),
	})
}

func (s *stacksServer) ClosePlan(ctx context.Context, req *stacks.CloseStackPlan_Request) (*stacks.CloseStackPlan_Response, error) {
	hnd := handle[*stackplan.Plan](req.PlanHandle)
	err := s.handles.CloseStackPlan(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &stacks.CloseStackPlan_Response{}, nil
}

func (s *stacksServer) ApplyStackChanges(req *stacks.ApplyStackChanges_Request, evts stacks.Stacks_ApplyStackChangesServer) error {
	ctx := evts.Context()
	syncEvts := newSyncStreamingRPCSender(evts)
	evts = nil // Prevent accidental unsynchronized usage of this server

	if req.PlanHandle != 0 && len(req.PlannedChanges) != 0 {
		return status.Error(codes.InvalidArgument, "must not set both plan_handle and planned_changes")
	}
	var plan *stackplan.Plan
	if req.PlanHandle != 0 {
		planHnd := handle[*stackplan.Plan](req.PlanHandle)
		plan = s.handles.StackPlan(planHnd)
		if plan == nil {
			return status.Error(codes.InvalidArgument, "the given plan handle is invalid")
		}
		// The plan handle is immediately invalidated by trying to apply it;
		// plans are not reusable because they are valid only against the
		// exact prior state they were generated for.
		if err := s.handles.CloseStackPlan(planHnd); err != nil {
			// It would be very strange to get here!
			return status.Error(codes.Internal, "failed to close the plan handle")
		}
	} else {
		// Deprecated: whole plan specified inline
		// FIXME: Remove this old field once our existing clients are updated.
		var err error
		plan, err = stackplan.LoadFromProto(req.PlannedChanges)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid planned_changes: %s", err)
		}
	}

	cfgHnd := handle[*stackconfig.Config](req.StackConfigHandle)
	cfg := s.handles.StackConfig(cfgHnd)
	if cfg == nil {
		return status.Error(codes.InvalidArgument, "the given stack configuration handle is invalid")
	}
	depsHnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	var deps *depsfile.Locks
	if s.providerDependencyLockOverride != nil {
		deps = s.providerDependencyLockOverride
	} else if !depsHnd.IsNil() {
		deps = s.handles.DependencyLocks(depsHnd)
		if deps == nil {
			return status.Error(codes.InvalidArgument, "the given dependency locks handle is invalid")
		}
	} else {
		deps = depsfile.NewLocks()
	}
	providerCacheHnd := handle[*providercache.Dir](req.ProviderCacheHandle)
	var providerCache *providercache.Dir
	if !providerCacheHnd.IsNil() {
		providerCache = s.handles.ProviderPluginCache(providerCacheHnd)
		if providerCache == nil {
			return status.Error(codes.InvalidArgument, "the given provider cache handle is invalid")
		}
	}
	var providerFactories map[addrs.Provider]providers.Factory
	if s.providerCacheOverride != nil {
		// This is only used in tests to side load providers without needing a
		// real provider cache.
		providerFactories = s.providerCacheOverride
	} else {
		// NOTE: providerCache can be nil if no handle was provided, in which
		// case the call can only use built-in providers. All code below
		// must avoid panicking when providerCache is nil, but is allowed to
		// return an InvalidArgument error in that case.
		// (providerFactoriesForLocks explicitly supports a nil providerCache)
		var err error
		// (providerFactoriesForLocks explicitly supports a nil providerCache)
		providerFactories, err = providerFactoriesForLocks(deps, providerCache)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "provider dependencies are inconsistent: %s", err)
		}
	}

	inputValues, err := externalInputValuesFromProto(req.InputValues)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid input values: %s", err)
	}

	// We'll hook some internal events in the planning process both to generate
	// tracing information if we're in an OpenTelemetry-aware context and
	// to propagate a subset of the events to our client.
	hooks := stackApplyHooks(syncEvts, cfg.Root.Stack.SourceAddr)
	ctx = stackruntime.ContextWithHooks(ctx, hooks)

	changesCh := make(chan stackstate.AppliedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	rtReq := stackruntime.ApplyRequest{
		Config:             cfg,
		InputValues:        inputValues,
		ProviderFactories:  providerFactories,
		Plan:               plan,
		ExperimentsAllowed: s.experimentsAllowed,
		DependencyLocks:    *deps,
	}
	rtResp := stackruntime.ApplyResponse{
		AppliedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	// As a long-running operation, the apply RPC must be able to be stopped.
	// We do this by requesting a stop channel from the stopper, and using it
	// to cancel the planning process.
	stopCh := s.stopper.add()
	defer s.stopper.remove(stopCh)

	// We create a new cancellable context for the stack plan operation to
	// allow us to respond to stop requests.
	applyCtx, cancelApply := context.WithCancel(ctx)
	defer cancelApply()

	// The actual apply operation runs in the background, and emits events
	// to us via the channels in rtResp before finally closing changesCh
	// to signal that the process is complete.
	go stackruntime.Apply(applyCtx, &rtReq, &rtResp)

	emitDiag := func(diag tfdiags.Diagnostic) {
		diags := tfdiags.Diagnostics{diag}
		protoDiags := diagnosticsToProto(diags)
		for _, protoDiag := range protoDiags {
			syncEvts.Send(&stacks.ApplyStackChanges_Event{
				Event: &stacks.ApplyStackChanges_Event_Diagnostic{
					Diagnostic: protoDiag,
				},
			})
		}
	}

	// There is no strong ordering between the planned changes and the
	// diagnostics, so we need to be prepared for them to arrive in any
	// order. However, stackruntime.Apply does guarantee that it will
	// close changesCh only after it's finished writing to and closing
	// everything else, and so we can assume that once changesCh is
	// closed we only need to worry about whatever's left in the
	// diagsCh buffer.
Events:
	for {
		select {

		case change, ok := <-changesCh:
			if !ok {
				if diagsCh != nil {
					// Async work is done! We do still need to consume the rest
					// of diagsCh before we stop, though, because there might
					// be some extras in the channel's buffer that we didn't
					// get to yet.
					for diag := range diagsCh {
						emitDiag(diag)
					}
				}
				break Events
			}

			protoChange, err := change.AppliedChangeProto()
			if err != nil {
				// Should not get here: it always indicates a bug in
				// AppliedChangeProto or in the code which constructed
				// the change over in package stackeval.
				// If we get here then it's likely that something will be
				// left stale in the final stack state, so we should really
				// avoid ever getting here.
				emitDiag(tfdiags.Sourceless(
					tfdiags.Error,
					"Incorrectly-constructed apply result",
					fmt.Sprintf(
						"Failed to serialize a %T value for recording in the updated state: %s.\n\nThis is a bug in Terraform; please report it!",
						protoChange, err,
					),
				))
				continue
			}

			syncEvts.Send(&stacks.ApplyStackChanges_Event{
				Event: &stacks.ApplyStackChanges_Event_AppliedChange{
					AppliedChange: protoChange,
				},
			})

		case diag, ok := <-diagsCh:
			if !ok {
				// The diagnostics channel has closed, so we'll just stop
				// trying to read from it and wait for changesCh to close,
				// which will be our final signal that everything is done.
				diagsCh = nil
				continue
			}
			emitDiag(diag)

		case <-stopCh:
			// If our stop channel is signalled, we need to cancel the apply.
			// This may result in remaining changes or diagnostics being
			// emitted, so we continue to monitor those channels if they're
			// still active.
			cancelApply()

		}
	}

	return nil
}

func (s *stacksServer) OpenStackInspector(ctx context.Context, req *stacks.OpenStackInspector_Request) (*stacks.OpenStackInspector_Response, error) {
	cfgHnd := handle[*stackconfig.Config](req.StackConfigHandle)
	cfg := s.handles.StackConfig(cfgHnd)
	if cfg == nil {
		return nil, status.Error(codes.InvalidArgument, "the given stack configuration handle is invalid")
	}
	depsHnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	var deps *depsfile.Locks
	if !depsHnd.IsNil() {
		deps = s.handles.DependencyLocks(depsHnd)
		if deps == nil {
			return nil, status.Error(codes.InvalidArgument, "the given dependency locks handle is invalid")
		}
	} else {
		deps = depsfile.NewLocks()
	}
	providerCacheHnd := handle[*providercache.Dir](req.ProviderCacheHandle)
	var providerCache *providercache.Dir
	if !providerCacheHnd.IsNil() {
		providerCache = s.handles.ProviderPluginCache(providerCacheHnd)
		if providerCache == nil {
			return nil, status.Error(codes.InvalidArgument, "the given provider cache handle is invalid")
		}
	}
	// NOTE: providerCache can be nil if no handle was provided, in which
	// case the call can only use built-in providers. All code below
	// must avoid panicking when providerCache is nil, but is allowed to
	// return an InvalidArgument error in that case.
	// (providerFactoriesForLocks explicitly supports a nil providerCache)
	providerFactories, err := providerFactoriesForLocks(deps, providerCache)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "provider dependencies are inconsistent: %s", err)
	}
	inputValues, err := externalInputValuesFromProto(req.InputValues)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid input values: %s", err)
	}
	state, err := stackstate.LoadFromProto(req.State)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "can't load state snapshot: %s", err)
	}

	hnd := s.handles.NewStackInspector(&stacksInspector{
		Config:             cfg,
		State:              state,
		ProviderFactories:  providerFactories,
		InputValues:        inputValues,
		ExperimentsAllowed: s.experimentsAllowed,
	})

	return &stacks.OpenStackInspector_Response{
		StackInspectorHandle: hnd.ForProtobuf(),
		// There are currently no situations that return diagnostics, but
		// we reserve the right to add some later.
	}, nil
}

func (s *stacksServer) ListResourceIdentities(ctx context.Context, req *stacks.ListResourceIdentities_Request) (*stacks.ListResourceIdentities_Response, error) {
	hnd := handle[*stackstate.State](req.StateHandle)
	stackState := s.handles.StackState(hnd)
	if stackState == nil {
		return nil, status.Error(codes.InvalidArgument, "the given stack state handle is invalid")
	}

	depsHnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	var deps *depsfile.Locks
	if !depsHnd.IsNil() {
		deps = s.handles.DependencyLocks(depsHnd)
		if deps == nil {
			return nil, status.Error(codes.InvalidArgument, "the given dependency locks handle is invalid")
		}
	} else {
		deps = depsfile.NewLocks()
	}
	providerCacheHnd := handle[*providercache.Dir](req.ProviderCacheHandle)
	var providerCache *providercache.Dir
	if !providerCacheHnd.IsNil() {
		providerCache = s.handles.ProviderPluginCache(providerCacheHnd)
		if providerCache == nil {
			return nil, status.Error(codes.InvalidArgument, "the given provider cache handle is invalid")
		}
	}
	// NOTE: providerCache can be nil if no handle was provided, in which
	// case the call can only use built-in providers. All code below
	// must avoid panicking when providerCache is nil, but is allowed to
	// return an InvalidArgument error in that case.
	// (providerFactoriesForLocks explicitly supports a nil providerCache)
	providerFactories, err := providerFactoriesForLocks(deps, providerCache)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "provider dependencies are inconsistent: %s", err)
	}

	identitySchemas := make(map[addrs.Provider]map[string]providers.IdentitySchema)
	for name, factory := range providerFactories {
		provider, err := factory()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "provider %s failed to initialize: %s", name, err)
		}

		schema := provider.GetResourceIdentitySchemas()
		if len(schema.Diagnostics) > 0 {
			return nil, status.Errorf(codes.InvalidArgument, "provider %s failed to retrieve schema: %s", name, schema.Diagnostics.Err())
		} else {
			identitySchemas[name] = schema.IdentityTypes
		}
	}

	resourceIdentities, err := listResourceIdentities(stackState, identitySchemas)
	if err != nil {
		return nil, err
	}

	return &stacks.ListResourceIdentities_Response{
		Resource: resourceIdentities,
	}, nil
}

func (s *stacksServer) InspectExpressionResult(ctx context.Context, req *stacks.InspectExpressionResult_Request) (*stacks.InspectExpressionResult_Response, error) {
	hnd := handle[*stacksInspector](req.StackInspectorHandle)
	insp := s.handles.StackInspector(hnd)
	if insp == nil {
		return nil, status.Error(codes.InvalidArgument, "the given stack inspector handle is invalid")
	}
	return insp.InspectExpressionResult(ctx, req)
}

func (s *stacksServer) OpenTerraformState(ctx context.Context, request *stacks.OpenTerraformState_Request) (*stacks.OpenTerraformState_Response, error) {
	switch data := request.State.(type) {
	case *stacks.OpenTerraformState_Request_ConfigPath:
		// Load the state from the backend.
		// This function should return an empty state even if the diags
		// has errors. This makes it easier for the caller, as they should
		// close the state handle regardless of the diags.
		loader := stackmigrate.Loader{Discovery: s.services}
		state, diags := loader.LoadState(data.ConfigPath)

		hnd := s.handles.NewTerraformState(state)
		return &stacks.OpenTerraformState_Response{
			StateHandle: hnd.ForProtobuf(),
			Diagnostics: diagnosticsToProto(diags),
		}, nil

	case *stacks.OpenTerraformState_Request_Raw:
		// load the state from the raw data
		file, err := statefile.Read(bytes.NewReader(data.Raw))
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid raw state data: %s", err)
		}

		hnd := s.handles.NewTerraformState(file.State)
		return &stacks.OpenTerraformState_Response{
			StateHandle: hnd.ForProtobuf(),
		}, nil

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid state source")
	}
}

func (s *stacksServer) CloseTerraformState(ctx context.Context, request *stacks.CloseTerraformState_Request) (*stacks.CloseTerraformState_Response, error) {
	hnd := handle[*states.State](request.StateHandle)
	err := s.handles.CloseTerraformState(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return new(stacks.CloseTerraformState_Response), nil
}

func (s *stacksServer) MigrateTerraformState(request *stacks.MigrateTerraformState_Request, server stacks.Stacks_MigrateTerraformStateServer) error {

	previousStateHandle := handle[*states.State](request.StateHandle)
	previousState := s.handles.TerraformState(previousStateHandle)
	if previousState == nil {
		return status.Error(codes.InvalidArgument, "the given state handle is invalid")
	}

	configHandle := handle[*stackconfig.Config](request.ConfigHandle)
	config := s.handles.StackConfig(configHandle)
	if config == nil {
		return status.Error(codes.InvalidArgument, "the given config handle is invalid")
	}

	dependencyLocksHandle := handle[*depsfile.Locks](request.DependencyLocksHandle)
	dependencyLocks := s.handles.DependencyLocks(dependencyLocksHandle)
	if dependencyLocks == nil {
		return status.Error(codes.InvalidArgument, "the given dependency locks handle is invalid")
	}

	var providerFactories map[addrs.Provider]providers.Factory
	if s.providerCacheOverride != nil {
		// This is only used in tests to side load providers without needing a
		// real provider cache.
		providerFactories = s.providerCacheOverride
	} else {
		providerCacheHandle := handle[*providercache.Dir](request.ProviderCacheHandle)
		providerCache := s.handles.ProviderPluginCache(providerCacheHandle)
		if providerCache == nil {
			return status.Error(codes.InvalidArgument, "the given provider cache handle is invalid")
		}

		var err error
		providerFactories, err = providerFactoriesForLocks(dependencyLocks, providerCache)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "provider dependencies are inconsistent: %s", err)
		}
	}

	migrate := &stackmigrate.Migration{
		Providers:     providerFactories,
		PreviousState: previousState,
		Config:        config,
	}

	emit := func(change stackstate.AppliedChange) {
		proto, err := change.AppliedChangeProto()
		if err != nil {
			server.Send(&stacks.MigrateTerraformState_Event{
				Result: &stacks.MigrateTerraformState_Event_Diagnostic{
					Diagnostic: &terraform1.Diagnostic{
						Severity: terraform1.Diagnostic_ERROR,
						Summary:  "Failed to serialize change",
						Detail:   fmt.Sprintf("Failed to serialize state change for recording in the migration plan: %s", err),
					},
				},
			})
			return
		}

		server.Send(&stacks.MigrateTerraformState_Event{
			Result: &stacks.MigrateTerraformState_Event_AppliedChange{
				AppliedChange: proto,
			},
		})
	}

	emitDiag := func(diagnostic tfdiags.Diagnostic) {
		server.Send(&stacks.MigrateTerraformState_Event{
			Result: &stacks.MigrateTerraformState_Event_Diagnostic{
				Diagnostic: diagnosticToProto(diagnostic),
			},
		})
	}

	mapping := request.GetMapping()
	if mapping == nil {
		return status.Error(codes.InvalidArgument, "missing migration mapping")
	}
	switch mapping := mapping.(type) {
	case *stacks.MigrateTerraformState_Request_Simple:
		migrate.Migrate(
			mapping.Simple.ResourceAddressMap,
			mapping.Simple.ModuleAddressMap,
			emit, emitDiag)
	default:
		return status.Error(codes.InvalidArgument, "unsupported migration mapping")
	}

	return nil
}

func stackPlanHooks(evts *syncPlanStackChangesServer, mainStackSource sourceaddrs.FinalSource) *stackruntime.Hooks {
	return stackChangeHooks(
		func(scp *stacks.StackChangeProgress) error {
			return evts.Send(&stacks.PlanStackChanges_Event{
				Event: &stacks.PlanStackChanges_Event_Progress{
					Progress: scp,
				},
			})
		},
		mainStackSource,
	)
}

func stackApplyHooks(evts *syncApplyStackChangesServer, mainStackSource sourceaddrs.FinalSource) *stackruntime.Hooks {
	return stackChangeHooks(
		func(scp *stacks.StackChangeProgress) error {
			return evts.Send(&stacks.ApplyStackChanges_Event{
				Event: &stacks.ApplyStackChanges_Event_Progress{
					Progress: scp,
				},
			})
		},
		mainStackSource,
	)
}

// stackChangeHooks is the shared hook-handling logic for both [stackPlanHooks]
// and [stackApplyHooks]. Each phase emits a different subset of the events
// handled here.
func stackChangeHooks(send func(*stacks.StackChangeProgress) error, mainStackSource sourceaddrs.FinalSource) *stackruntime.Hooks {
	return &stackruntime.Hooks{
		// For any BeginFunc-shaped hook that returns an OpenTelemetry tracing
		// span, we'll wrap it in a context so that the runtime's downstream
		// operations will appear as children of it.
		ContextAttach: func(parent context.Context, tracking any) context.Context {
			span, ok := tracking.(trace.Span)
			if !ok {
				return parent
			}
			return trace.ContextWithSpan(parent, span)
		},

		// For the overall plan operation we don't emit any events to the client,
		// since it already knows it has asked us to plan, but we do establish
		// a root tracing span for all of the downstream planning operations to
		// attach themselves to.
		BeginPlan: func(ctx context.Context, s struct{}) any {
			_, span := tracer.Start(ctx, "planning", trace.WithAttributes(
				attribute.String("main_stack_source", mainStackSource.String()),
			))
			return span
		},
		EndPlan: func(ctx context.Context, span any, s struct{}) any {
			span.(trace.Span).End()
			return nil
		},

		// For the overall apply operation we don't emit any events to the client,
		// since it already knows it has asked us to apply, but we do establish
		// a root tracing span for all of the downstream planning operations to
		// attach themselves to.
		BeginApply: func(ctx context.Context, s struct{}) any {
			_, span := tracer.Start(ctx, "applying", trace.WithAttributes(
				attribute.String("main_stack_source", mainStackSource.String()),
			))
			return span
		},
		EndApply: func(ctx context.Context, span any, s struct{}) any {
			span.(trace.Span).End()
			return nil
		},

		// After expanding a component, we emit an event to the client to
		// list all of the resulting instances. In the common case of an
		// unexpanded component, this will be a single address.
		ComponentExpanded: func(ctx context.Context, ce *hooks.ComponentInstances) {
			ias := make([]string, 0, len(ce.InstanceAddrs))
			for _, ia := range ce.InstanceAddrs {
				ias = append(ias, ia.String())
			}
			send(&stacks.StackChangeProgress{
				Event: &stacks.StackChangeProgress_ComponentInstances_{
					ComponentInstances: &stacks.StackChangeProgress_ComponentInstances{
						ComponentAddr: ce.ComponentAddr.String(),
						InstanceAddrs: ias,
					},
				},
			})
		},

		// For each component instance, we emit a series of events to the
		// client, reporting the status of the plan operation. We also create a
		// nested tracing span for the component instance.
		PendingComponentInstancePlan: func(ctx context.Context, ci stackaddrs.AbsComponentInstance) {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstancePending))
		},
		BeginComponentInstancePlan: func(ctx context.Context, ci stackaddrs.AbsComponentInstance) any {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstancePlanning))
			_, span := tracer.Start(ctx, "planning", trace.WithAttributes(
				attribute.String("component_instance", ci.String()),
			))
			return span
		},
		EndComponentInstancePlan: func(ctx context.Context, span any, ci stackaddrs.AbsComponentInstance) any {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstancePlanned))
			span.(trace.Span).SetStatus(otelCodes.Ok, "planning succeeded")
			span.(trace.Span).End()
			return nil
		},
		ErrorComponentInstancePlan: func(ctx context.Context, span any, ci stackaddrs.AbsComponentInstance) any {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstanceErrored))
			span.(trace.Span).SetStatus(otelCodes.Error, "planning failed")
			span.(trace.Span).End()
			return nil
		},
		DeferComponentInstancePlan: func(ctx context.Context, span any, ci stackaddrs.AbsComponentInstance) any {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstanceDeferred))
			span.(trace.Span).SetStatus(otelCodes.Error, "planning succeeded, but deferred")
			span.(trace.Span).End()
			return nil
		},
		PendingComponentInstanceApply: func(ctx context.Context, ci stackaddrs.AbsComponentInstance) {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstancePending))
		},
		BeginComponentInstanceApply: func(ctx context.Context, ci stackaddrs.AbsComponentInstance) any {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstanceApplying))
			_, span := tracer.Start(ctx, "applying", trace.WithAttributes(
				attribute.String("component_instance", ci.String()),
			))
			return span
		},
		EndComponentInstanceApply: func(ctx context.Context, span any, ci stackaddrs.AbsComponentInstance) any {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstanceApplied))
			span.(trace.Span).SetStatus(otelCodes.Ok, "applying succeeded")
			span.(trace.Span).End()
			return nil
		},
		ErrorComponentInstanceApply: func(ctx context.Context, span any, ci stackaddrs.AbsComponentInstance) any {
			send(evtComponentInstanceStatus(ci, hooks.ComponentInstanceErrored))
			span.(trace.Span).SetStatus(otelCodes.Error, "applying failed")
			span.(trace.Span).End()
			return nil
		},

		// When Terraform core reports a resource instance plan status, we
		// forward it to the events client.
		ReportResourceInstanceStatus: func(ctx context.Context, span any, rihd *hooks.ResourceInstanceStatusHookData) any {
			// addrs.Provider.String() will panic on the zero value. In this
			// case, holding a zero provider would mean a bug in our event
			// logging code rather than in core logic, so avoid exploding, but
			// send a blank string to expose the error later.
			providerAddr := ""
			if !rihd.ProviderAddr.IsZero() {
				providerAddr = rihd.ProviderAddr.String()
			}
			send(&stacks.StackChangeProgress{
				Event: &stacks.StackChangeProgress_ResourceInstanceStatus_{
					ResourceInstanceStatus: &stacks.StackChangeProgress_ResourceInstanceStatus{
						Addr:         stacks.NewResourceInstanceObjectInStackAddr(rihd.Addr),
						Status:       rihd.Status.ForProtobuf(),
						ProviderAddr: providerAddr,
					},
				},
			})
			return span
		},

		// Upon completion of a component instance plan, we emit a planned
		// change sumary event to the client for each resource instance.
		ReportResourceInstancePlanned: func(ctx context.Context, span any, ric *hooks.ResourceInstanceChange) any {
			span.(trace.Span).AddEvent("planned resource instance", trace.WithAttributes(
				attribute.String("component_instance", ric.Addr.Component.String()),
				attribute.String("resource_instance", ric.Addr.Item.String()),
			))

			ripc, err := resourceInstancePlanned(ric)
			if err != nil {
				return span
			}

			send(&stacks.StackChangeProgress{
				Event: &stacks.StackChangeProgress_ResourceInstancePlannedChange_{
					ResourceInstancePlannedChange: ripc,
				},
			})
			return span
		},

		ReportResourceInstanceDeferred: func(ctx context.Context, span any, change *hooks.DeferredResourceInstanceChange) any {
			span.(trace.Span).AddEvent("deferred resource instance", trace.WithAttributes(
				attribute.String("component_instance", change.Change.Addr.Component.String()),
				attribute.String("resource_instance", change.Change.Addr.Item.String()),
			))

			ripc, err := resourceInstancePlanned(change.Change)
			if err != nil {
				return span
			}

			deferred := stackplan.EncodeDeferred(change.Reason)

			send(&stacks.StackChangeProgress{
				Event: &stacks.StackChangeProgress_DeferredResourceInstancePlannedChange_{
					DeferredResourceInstancePlannedChange: &stacks.StackChangeProgress_DeferredResourceInstancePlannedChange{
						Change:   ripc,
						Deferred: deferred,
					},
				},
			})
			return span
		},

		// We also report a roll-up of planned resource action counts after each
		// component instance plan or apply completes.
		ReportComponentInstancePlanned: func(ctx context.Context, span any, cic *hooks.ComponentInstanceChange) any {
			send(&stacks.StackChangeProgress{
				Event: &stacks.StackChangeProgress_ComponentInstanceChanges_{
					ComponentInstanceChanges: &stacks.StackChangeProgress_ComponentInstanceChanges{
						Addr: &stacks.ComponentInstanceInStackAddr{
							ComponentAddr:         stackaddrs.ConfigComponentForAbsInstance(cic.Addr).String(),
							ComponentInstanceAddr: cic.Addr.String(),
						},
						Total:  int32(cic.Total()),
						Add:    int32(cic.Add),
						Change: int32(cic.Change),
						Import: int32(cic.Import),
						Remove: int32(cic.Remove),
						Defer:  int32(cic.Defer),
						Move:   int32(cic.Move),
						Forget: int32(cic.Forget),
					},
				},
			})
			return span
		},
		// The apply rollup should typically report the same information as
		// the plan one did earlier, but could vary in some situations if
		// e.g. a planned update turned out to be a no-op once some unknown
		// values were known, or if the apply phase is handled by a different
		// version of the agent than the plan phase which has support for
		// a different set of possible change types.
		ReportComponentInstanceApplied: func(ctx context.Context, span any, cic *hooks.ComponentInstanceChange) any {
			send(&stacks.StackChangeProgress{
				Event: &stacks.StackChangeProgress_ComponentInstanceChanges_{
					ComponentInstanceChanges: &stacks.StackChangeProgress_ComponentInstanceChanges{
						Addr: &stacks.ComponentInstanceInStackAddr{
							ComponentAddr:         stackaddrs.ConfigComponentForAbsInstance(cic.Addr).String(),
							ComponentInstanceAddr: cic.Addr.String(),
						},
						Total:  int32(cic.Total()),
						Add:    int32(cic.Add),
						Change: int32(cic.Change),
						Import: int32(cic.Import),
						Remove: int32(cic.Remove),
						Defer:  int32(cic.Defer),
						Move:   int32(cic.Move),
						Forget: int32(cic.Forget),
					},
				},
			})
			return span
		},
	}
}

func resourceInstancePlanned(ric *hooks.ResourceInstanceChange) (*stacks.StackChangeProgress_ResourceInstancePlannedChange, error) {
	actions, err := stacks.ChangeTypesForPlanAction(ric.Change.Action)
	if err != nil {
		return nil, err
	}

	var moved *stacks.StackChangeProgress_ResourceInstancePlannedChange_Moved
	if !ric.Change.PrevRunAddr.Equal(ric.Change.Addr) {
		moved = &stacks.StackChangeProgress_ResourceInstancePlannedChange_Moved{
			PrevAddr: &stacks.ResourceInstanceInStackAddr{
				ComponentInstanceAddr: ric.Addr.Component.String(),
				ResourceInstanceAddr:  ric.Change.PrevRunAddr.String(),
			},
		}
	}

	var imported *stacks.StackChangeProgress_ResourceInstancePlannedChange_Imported
	if ric.Change.Importing != nil {
		imported = &stacks.StackChangeProgress_ResourceInstancePlannedChange_Imported{
			ImportId: ric.Change.Importing.ID,
			Unknown:  ric.Change.Importing.Unknown,
		}
	}

	return &stacks.StackChangeProgress_ResourceInstancePlannedChange{
		Addr:         stacks.NewResourceInstanceObjectInStackAddr(ric.Addr),
		Actions:      actions,
		Moved:        moved,
		Imported:     imported,
		ProviderAddr: ric.Change.ProviderAddr.Provider.String(),
	}, nil
}

func evtComponentInstanceStatus(ci stackaddrs.AbsComponentInstance, status hooks.ComponentInstanceStatus) *stacks.StackChangeProgress {
	return &stacks.StackChangeProgress{
		Event: &stacks.StackChangeProgress_ComponentInstanceStatus_{
			ComponentInstanceStatus: &stacks.StackChangeProgress_ComponentInstanceStatus{
				Addr: &stacks.ComponentInstanceInStackAddr{
					ComponentAddr:         stackaddrs.ConfigComponentForAbsInstance(ci).String(),
					ComponentInstanceAddr: ci.String(),
				},
				Status: status.ForProtobuf(),
			},
		},
	}
}

// syncPlanStackChangesServer is a wrapper around a
// stacks.Stacks_PlanStackChangesServer implementation that makes the
// Send method concurrency-safe by holding a mutex throughout the underlying
// call.
type syncPlanStackChangesServer = syncStreamingRPCSender[stacks.Stacks_PlanStackChangesServer, *stacks.PlanStackChanges_Event]

// syncApplyStackChangesServer is a wrapper around a
// stacks.Stacks_ApplyStackChangesServer implementation that makes the
// Send method concurrency-safe by holding a mutex throughout the underlying
// call.
type syncApplyStackChangesServer = syncStreamingRPCSender[stacks.Stacks_ApplyStackChangesServer, *stacks.ApplyStackChanges_Event]
