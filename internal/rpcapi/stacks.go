package rpcapi

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type stacksServer struct {
	terraform1.UnimplementedStacksServer

	handles *handleTable
}

var _ terraform1.StacksServer = (*stacksServer)(nil)

func newStacksServer(handles *handleTable) *stacksServer {
	return &stacksServer{
		handles: handles,
	}
}

func (s *stacksServer) OpenStackConfiguration(ctx context.Context, req *terraform1.OpenStackConfiguration_Request) (*terraform1.OpenStackConfiguration_Response, error) {
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
		return &terraform1.OpenStackConfiguration_Response{
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
	return &terraform1.OpenStackConfiguration_Response{
		StackConfigHandle: configHnd.ForProtobuf(),
		Diagnostics:       diagnosticsToProto(diags),
	}, nil
}

func (s *stacksServer) CloseStackConfiguration(ctx context.Context, req *terraform1.CloseStackConfiguration_Request) (*terraform1.CloseStackConfiguration_Response, error) {
	hnd := handle[*stackconfig.Config](req.StackConfigHandle)
	err := s.handles.CloseStackConfig(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &terraform1.CloseStackConfiguration_Response{}, nil
}

func (s *stacksServer) FindStackConfigurationComponents(ctx context.Context, req *terraform1.FindStackConfigurationComponents_Request) (*terraform1.FindStackConfigurationComponents_Response, error) {
	cfgHnd := handle[*stackconfig.Config](req.StackConfigHandle)
	cfg := s.handles.StackConfig(cfgHnd)
	if cfg == nil {
		return nil, status.Error(codes.InvalidArgument, "the given stack configuration handle is invalid")
	}

	return &terraform1.FindStackConfigurationComponents_Response{
		Config: stackConfigMetaforProto(cfg.Root),
	}, nil
}

func stackConfigMetaforProto(cfgNode *stackconfig.ConfigNode) *terraform1.FindStackConfigurationComponents_StackConfig {
	ret := &terraform1.FindStackConfigurationComponents_StackConfig{
		Components:     make(map[string]*terraform1.FindStackConfigurationComponents_Component),
		EmbeddedStacks: make(map[string]*terraform1.FindStackConfigurationComponents_EmbeddedStack),
	}

	for name, cc := range cfgNode.Stack.Components {
		cProto := &terraform1.FindStackConfigurationComponents_Component{
			SourceAddr: cc.FinalSourceAddr.String(),
		}
		switch {
		case cc.ForEach != nil:
			cProto.Instances = terraform1.FindStackConfigurationComponents_FOR_EACH
		default:
			cProto.Instances = terraform1.FindStackConfigurationComponents_SINGLE
		}
		ret.Components[name] = cProto
	}

	for name, sn := range cfgNode.Children {
		sc := cfgNode.Stack.EmbeddedStacks[name]
		sProto := &terraform1.FindStackConfigurationComponents_EmbeddedStack{
			SourceAddr: sn.Stack.SourceAddr.String(),
			Config:     stackConfigMetaforProto(sn),
		}
		switch {
		case sc.ForEach != nil:
			sProto.Instances = terraform1.FindStackConfigurationComponents_FOR_EACH
		default:
			sProto.Instances = terraform1.FindStackConfigurationComponents_SINGLE
		}
		ret.EmbeddedStacks[name] = sProto
	}

	return ret
}

func (s *stacksServer) PlanStackChanges(req *terraform1.PlanStackChanges_Request, evts terraform1.Stacks_PlanStackChangesServer) error {
	ctx := evts.Context()
	syncEvts := &syncPlanStackChangesServer{evts: evts}
	evts = nil // Prevent accidental unsynchronized usage of this server

	cfgHnd := handle[*stackconfig.Config](req.StackConfigHandle)
	cfg := s.handles.StackConfig(cfgHnd)
	if cfg == nil {
		return status.Error(codes.InvalidArgument, "the given stack configuration handle is invalid")
	}
	depsHnd := handle[*depsfile.Locks](req.DependencyLocksHandle)
	var deps *depsfile.Locks
	if !depsHnd.IsNil() {
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
	} else {
		// NOTE: providerCache can be nil if no handle was provided, in which
		// case the call can only use built-in providers. All code below
		// must avoid panicking when providerCache is nil, but is allowed to
		// return an InvalidArgument error in that case.
	}

	inputValues, err := externalInputValuesFromProto(req.InputValues)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid input values: %s", err)
	}

	// (providerFactoriesForLocks explicitly supports a nil providerCache)
	providerFactories, err := providerFactoriesForLocks(deps, providerCache)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "provider dependencies are inconsistent: %s", err)
	}

	// We'll hook some internal events in the planning process both to generate
	// tracing information if we're in an OpenTelemetry-aware context and
	// to propagate a subset of the events to our client.
	hooks := stackPlanHooks(syncEvts, cfg.Root.Stack.SourceAddr)
	ctx = stackruntime.ContextWithHooks(ctx, hooks)

	var planMode plans.Mode
	switch req.PlanMode {
	case terraform1.PlanMode_NORMAL:
		planMode = plans.NormalMode
	case terraform1.PlanMode_REFRESH_ONLY:
		planMode = plans.RefreshOnlyMode
	case terraform1.PlanMode_DESTROY:
		planMode = plans.DestroyMode
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported planning mode %d", req.PlanMode)
	}
	log.Printf("[TRACE] plan mode is %s", planMode) // TODO: Just so planMode is used for now

	if len(req.PreviousState) != 0 {
		// TODO: We don't yet support planning from a prior state.
		return status.Errorf(codes.InvalidArgument, "don't yet support planning with a previous state")
	}

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	rtReq := stackruntime.PlanRequest{
		Config:            cfg,
		ProviderFactories: providerFactories,
		InputValues:       inputValues,
	}
	rtResp := stackruntime.PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	// The actual plan operation runs in the background, and emits events
	// to us via the channels in rtResp before finally closing changesCh
	// to signal that the process is complete.
	go stackruntime.Plan(ctx, &rtReq, &rtResp)

	emitDiag := func(diag tfdiags.Diagnostic) {
		diags := tfdiags.Diagnostics{diag}
		protoDiags := diagnosticsToProto(diags)
		for _, protoDiag := range protoDiags {
			syncEvts.Send(&terraform1.PlanStackChanges_Event{
				Event: &terraform1.PlanStackChanges_Event_Diagnostic{
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

			syncEvts.Send(&terraform1.PlanStackChanges_Event{
				Event: &terraform1.PlanStackChanges_Event_PlannedChange{
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

		}
	}

	return nil
}

func stackPlanHooks(evts *syncPlanStackChangesServer, mainStackSource sourceaddrs.FinalSource) *stackruntime.Hooks {
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

		// For each component instance, we emit a series of events to the
		// client, reporting the status of the plan operation. We also create a
		// nested tracing span for the component instance.
		PendingComponentInstancePlan: func(ctx context.Context, ci stackaddrs.AbsComponentInstance) {
			evts.Send(evtComponentInstanceStatus(ci, hooks.ComponentInstancePending))
		},
		BeginComponentInstancePlan: func(ctx context.Context, ci stackaddrs.AbsComponentInstance) any {
			evts.Send(evtComponentInstanceStatus(ci, hooks.ComponentInstancePlanning))
			_, span := tracer.Start(ctx, "planning", trace.WithAttributes(
				attribute.String("component_instance", ci.String()),
			))
			return span
		},
		EndComponentInstancePlan: func(ctx context.Context, span any, ci stackaddrs.AbsComponentInstance) any {
			evts.Send(evtComponentInstanceStatus(ci, hooks.ComponentInstancePlanned))
			span.(trace.Span).End()
			return nil
		},
		ErrorComponentInstancePlan: func(ctx context.Context, span any, ci stackaddrs.AbsComponentInstance) any {
			evts.Send(evtComponentInstanceStatus(ci, hooks.ComponentInstanceErrored))
			span.(trace.Span).End()
			return nil
		},

		// When Terraform core reports a resource instance plan status, we
		// forward it to the events client.
		ReportResourceInstanceStatus: func(ctx context.Context, span any, rihd *hooks.ResourceInstanceStatusHookData) any {
			evts.Send(&terraform1.PlanStackChanges_Event{
				Event: &terraform1.PlanStackChanges_Event_ResourceInstanceStatus{
					ResourceInstanceStatus: &terraform1.ResourceInstanceStatus{
						Addr: &terraform1.ResourceInstanceInStackAddr{
							ComponentInstanceAddr: rihd.Addr.Component.String(),
							ResourceInstanceAddr:  rihd.Addr.Item.String(),
						},
						Status: rihd.Status.ForProtobuf(),
					},
				},
			})
			return span
		},

		// Upon completion of a component instance plan, we emit a planned
		// change sumary event to the client for each resource instance.
		ReportResourceInstancePlanned: func(ctx context.Context, span any, ric *hooks.ResourceInstanceChange) any {
			actions, err := terraform1.ChangeTypesForPlanAction(ric.Change.Action)
			if err != nil {
				// TODO: what do we do?
				return span
			}

			moved := &terraform1.ResourceInstancePlannedChange_Moved{}
			if !ric.Change.PrevRunAddr.Equal(ric.Change.Addr) {
				moved.PrevAddr = &terraform1.ResourceInstanceInStackAddr{
					ComponentInstanceAddr: ric.Addr.Component.String(),
					ResourceInstanceAddr:  ric.Change.PrevRunAddr.String(),
				}
			}

			imported := &terraform1.ResourceInstancePlannedChange_Imported{}
			if ric.Change.Importing != nil {
				imported.ImportId = ric.Change.Importing.ID
			}

			evts.Send(&terraform1.PlanStackChanges_Event{
				Event: &terraform1.PlanStackChanges_Event_ResourceInstancePlannedChange{
					ResourceInstancePlannedChange: &terraform1.ResourceInstancePlannedChange{
						Addr: &terraform1.ResourceInstanceInStackAddr{
							ComponentInstanceAddr: ric.Addr.Component.String(),
							ResourceInstanceAddr:  ric.Addr.Item.String(),
						},
						Actions:  actions,
						Moved:    moved,
						Imported: imported,
					},
				},
			})
			return span
		},
	}
}

func evtComponentInstanceStatus(ci stackaddrs.AbsComponentInstance, status hooks.ComponentInstanceStatus) *terraform1.PlanStackChanges_Event {
	return &terraform1.PlanStackChanges_Event{
		Event: &terraform1.PlanStackChanges_Event_ComponentInstanceStatus{
			ComponentInstanceStatus: &terraform1.ComponentInstanceStatus{
				Addr: &terraform1.ComponentInstanceInStackAddr{
					ComponentAddr:         stackaddrs.ConfigComponentForAbsInstance(ci).String(),
					ComponentInstanceAddr: ci.String(),
				},
				Status: status.ForProtobuf(),
			},
		},
	}
}

// syncPlanStackChangesServer is a wrapper around the gprc.ServerStream
// instance used for planning events. This is required because the underlying
// grpc server is not concurrency safe on send.
//
// TODO: consider making this generic over multiple grpc server types.
type syncPlanStackChangesServer struct {
	evts terraform1.Stacks_PlanStackChangesServer
	mu   sync.Mutex
}

func (s *syncPlanStackChangesServer) Send(evt *terraform1.PlanStackChanges_Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.evts.Send(evt)
}
