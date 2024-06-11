// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform-svchost/disco"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/version"
)

func TestStacksOpenCloseStackConfiguration(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	stacksServer := newStacksServer(newStopper(), handles, &serviceOpts{})

	// In normal use a client would have previously opened a source bundle
	// using Dependencies.OpenSourceBundle, so we'll simulate the effect
	// of that here.
	var sourcesHnd handle[*sourcebundle.Bundle]
	{
		sources, err := sourcebundle.OpenDir("testdata/sourcebundle")
		if err != nil {
			t.Fatal(err)
		}
		sourcesHnd = handles.NewSourceBundle(sources)
	}

	openResp, err := stacksServer.OpenStackConfiguration(ctx, &terraform1.OpenStackConfiguration_Request{
		SourceBundleHandle: sourcesHnd.ForProtobuf(),
		SourceAddress: &terraform1.SourceAddress{
			Source: "git::https://example.com/foo.git",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// A client wouldn't normally be able to interact directly with the
	// stack configuration, but we're doing that here to simulate what would
	// happen in another service that takes configuration handles as input.
	{
		hnd := handle[*stackconfig.Config](openResp.StackConfigHandle)
		cfg := handles.StackConfig(hnd)
		if cfg == nil {
			t.Fatal("returned stack config handle is invalid")
		}
	}

	// A hypothetical attempt to close the underlying source bundle while
	// the stack configuration is active should fail.
	{
		depsServer := newDependenciesServer(handles, disco.New())

		_, err := depsServer.CloseSourceBundle(ctx, &terraform1.CloseSourceBundle_Request{
			SourceBundleHandle: sourcesHnd.ForProtobuf(),
		})
		if err == nil {
			t.Fatal("successfully closed source bundle while stack config was using it; should have failed to close")
		}
		protoStatus, ok := status.FromError(err)
		if !ok {
			t.Fatal("error is not a protobuf status code")
		}
		if got, want := protoStatus.Code(), codes.InvalidArgument; got != want {
			t.Errorf("wrong error status\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := protoStatus.Message(), "handle is in use by another open handle"; got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	}

	_, err = stacksServer.CloseStackConfiguration(ctx, &terraform1.CloseStackConfiguration_Request{
		StackConfigHandle: openResp.StackConfigHandle,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should be able to close the source bundle now too.
	{
		depsServer := newDependenciesServer(handles, disco.New())

		_, err := depsServer.CloseSourceBundle(ctx, &terraform1.CloseSourceBundle_Request{
			SourceBundleHandle: sourcesHnd.ForProtobuf(),
		})
		if err != nil {
			t.Fatalf("failed to close the source bundle: %s", err.Error())
		}
	}
}

func TestStacksFindStackConfigurationComponents(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	stacksServer := newStacksServer(newStopper(), handles, &serviceOpts{})

	// In normal use a client would have previously opened a source bundle
	// using Dependencies.OpenSourceBundle, so we'll simulate the effect
	// of that here.
	var sourcesHnd handle[*sourcebundle.Bundle]
	{
		sources, err := sourcebundle.OpenDir("testdata/sourcebundle")
		if err != nil {
			t.Fatal(err)
		}
		sourcesHnd = handles.NewSourceBundle(sources)
	}

	t.Run("empty config", func(t *testing.T) {
		openResp, err := stacksServer.OpenStackConfiguration(ctx, &terraform1.OpenStackConfiguration_Request{
			SourceBundleHandle: sourcesHnd.ForProtobuf(),
			SourceAddress: &terraform1.SourceAddress{
				Source: "git::https://example.com/foo.git",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(openResp.Diagnostics) != 0 {
			t.Error("empty configuration generated diagnostics; expected none")
			if openResp.StackConfigHandle == 0 {
				return // Our later operations will fail if given the nil handle
			}
		}

		cmpntResp, err := stacksServer.FindStackConfigurationComponents(ctx, &terraform1.FindStackConfigurationComponents_Request{
			StackConfigHandle: openResp.StackConfigHandle,
		})
		if err != nil {
			t.Fatal(err)
		}

		got := cmpntResp.Config
		want := &terraform1.FindStackConfigurationComponents_StackConfig{
			// Intentionally empty, because the configuration we've loaded
			// is itself empty.
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("non-empty config", func(t *testing.T) {
		openResp, err := stacksServer.OpenStackConfiguration(ctx, &terraform1.OpenStackConfiguration_Request{
			SourceBundleHandle: sourcesHnd.ForProtobuf(),
			SourceAddress: &terraform1.SourceAddress{
				Source: "git::https://example.com/foo.git//non-empty-stack",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(openResp.Diagnostics) != 0 {
			t.Error("empty configuration generated diagnostics; expected none")
			if openResp.StackConfigHandle == 0 {
				return // Our later operations will fail if given the nil handle
			}
		}

		cmpntResp, err := stacksServer.FindStackConfigurationComponents(ctx, &terraform1.FindStackConfigurationComponents_Request{
			StackConfigHandle: openResp.StackConfigHandle,
		})
		if err != nil {
			t.Fatal(err)
		}

		got := cmpntResp.Config
		want := &terraform1.FindStackConfigurationComponents_StackConfig{
			Components: map[string]*terraform1.FindStackConfigurationComponents_Component{
				"single": {
					SourceAddr:    "git::https://example.com/foo.git//non-empty-stack/empty-module",
					ComponentAddr: "component.single",
				},
				"for_each": {
					SourceAddr:    "git::https://example.com/foo.git//non-empty-stack/empty-module",
					Instances:     terraform1.FindStackConfigurationComponents_FOR_EACH,
					ComponentAddr: "component.for_each",
				},
			},
			EmbeddedStacks: map[string]*terraform1.FindStackConfigurationComponents_EmbeddedStack{
				"single": {
					SourceAddr: "git::https://example.com/foo.git//non-empty-stack/child",
					Config: &terraform1.FindStackConfigurationComponents_StackConfig{
						Components: map[string]*terraform1.FindStackConfigurationComponents_Component{
							"foo": {
								SourceAddr:    "git::https://example.com/foo.git//non-empty-stack/empty-module",
								ComponentAddr: "stack.single.component.foo",
							},
						},
					},
				},
				"for_each": {
					SourceAddr: "git::https://example.com/foo.git//non-empty-stack/child",
					Instances:  terraform1.FindStackConfigurationComponents_FOR_EACH,
					Config: &terraform1.FindStackConfigurationComponents_StackConfig{
						Components: map[string]*terraform1.FindStackConfigurationComponents_Component{
							"foo": {
								SourceAddr:    "git::https://example.com/foo.git//non-empty-stack/empty-module",
								ComponentAddr: "stack.for_each.component.foo",
							},
						},
					},
				},
			},
			InputVariables: map[string]*terraform1.FindStackConfigurationComponents_InputVariable{
				"unused": {
					Optional: false,
				},
				"unused_with_default": {
					Optional: true,
				},
			},
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}

func TestStacksPlanStackChanges(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	stacksServer := newStacksServer(newStopper(), handles, &serviceOpts{})

	fakeSourceBundle := &sourcebundle.Bundle{}
	bundleHnd := handles.NewSourceBundle(fakeSourceBundle)
	emptyConfig := &stackconfig.Config{
		Root: &stackconfig.ConfigNode{
			Stack: &stackconfig.Stack{
				SourceAddr: sourceaddrs.MustParseSource("git::https://example.com/foo.git").(sourceaddrs.RemoteSource),
			},
		},
	}
	configHnd, err := handles.NewStackConfig(emptyConfig, bundleHnd)
	if err != nil {
		t.Fatal(err)
	}

	grpcClient, close := grpcClientForTesting(ctx, t, func(srv *grpc.Server) {
		terraform1.RegisterStacksServer(srv, stacksServer)
	})
	defer close()

	stacksClient := terraform1.NewStacksClient(grpcClient)
	events, err := stacksClient.PlanStackChanges(ctx, &terraform1.PlanStackChanges_Request{
		PlanMode:          terraform1.PlanMode_NORMAL,
		StackConfigHandle: configHnd.ForProtobuf(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	wantEvents := splitStackOperationEvents([]*terraform1.PlanStackChanges_Event{
		{
			Event: &terraform1.PlanStackChanges_Event_PlannedChange{
				PlannedChange: &terraform1.PlannedChange{
					Raw: []*anypb.Any{
						mustMarshalAnyPb(&tfstackdata1.PlanHeader{
							TerraformVersion: version.SemVer.String(),
						}),
					},
				},
			},
		},
		{
			Event: &terraform1.PlanStackChanges_Event_PlannedChange{
				PlannedChange: &terraform1.PlannedChange{
					Raw: []*anypb.Any{
						mustMarshalAnyPb(&tfstackdata1.PlanApplyable{
							Applyable: true,
						}),
					},
					Descriptions: []*terraform1.PlannedChange_ChangeDescription{
						{
							Description: &terraform1.PlannedChange_ChangeDescription_PlanApplyable{
								PlanApplyable: true,
							},
						},
					},
				},
			},
		},
	})
	var gotEventsAll []*terraform1.PlanStackChanges_Event
	for {
		event, err := events.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		gotEventsAll = append(gotEventsAll, event)
	}
	gotEvents := splitStackOperationEvents(gotEventsAll)

	if diff := cmp.Diff(wantEvents, gotEvents, protocmp.Transform()); diff != "" {
		t.Errorf("wrong events\n%s", diff)
	}
}

func TestStacksPlanStackChanges_DeferredChanges(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	stacksServer := newStacksServer(newStopper(), handles, &serviceOpts{})

	// For this test, we do actually want to use a "real" provider. We'll
	// use the providerCacheOverride to side-load the testing provider.
	stacksServer.providerCacheOverride = make(map[addrs.Provider]providers.Factory)
	stacksServer.providerCacheOverride[addrs.NewDefaultProvider("testing")] = func() (providers.Interface, error) {
		return stacks_testing_provider.NewProvider(), nil
	}

	// Enable deferrals for this test.
	stacksServer.deferralAllowed = true

	sb, err := sourcebundle.OpenDir("testdata/sourcebundle")
	if err != nil {
		t.Fatal(err)
	}
	hnd := handles.NewSourceBundle(sb)

	client, close := grpcClientForTesting(ctx, t, func(srv *grpc.Server) {
		terraform1.RegisterStacksServer(srv, stacksServer)
	})
	defer close()

	stacks := terraform1.NewStacksClient(client)

	open, err := stacks.OpenStackConfiguration(ctx, &terraform1.OpenStackConfiguration_Request{
		SourceBundleHandle: hnd.ForProtobuf(),
		SourceAddress: &terraform1.SourceAddress{
			Source: "git::https://example.com/bar.git",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer stacks.CloseStackConfiguration(ctx, &terraform1.CloseStackConfiguration_Request{
		StackConfigHandle: open.StackConfigHandle,
	})

	resp, err := stacks.PlanStackChanges(ctx, &terraform1.PlanStackChanges_Request{
		PlanMode:          terraform1.PlanMode_NORMAL,
		StackConfigHandle: open.StackConfigHandle,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	wantEvents := splitStackOperationEvents([]*terraform1.PlanStackChanges_Event{
		// We're checking for the progress events for the deferred changes
		// here.
		{
			Event: &terraform1.PlanStackChanges_Event_Progress{
				Progress: &terraform1.StackChangeProgress{
					Event: &terraform1.StackChangeProgress_ComponentInstanceChanges_{
						ComponentInstanceChanges: &terraform1.StackChangeProgress_ComponentInstanceChanges{
							Addr: &terraform1.ComponentInstanceInStackAddr{
								ComponentAddr:         "component.deferred",
								ComponentInstanceAddr: "component.deferred",
							},
							Total:    1,
							Deferred: 1,
						},
					},
				},
			},
		},
		{
			Event: &terraform1.PlanStackChanges_Event_Progress{
				Progress: &terraform1.StackChangeProgress{
					Event: &terraform1.StackChangeProgress_DeferredResourceInstancePlannedChange_{
						DeferredResourceInstancePlannedChange: &terraform1.StackChangeProgress_DeferredResourceInstancePlannedChange{
							Deferred: &terraform1.Deferred{
								Reason: terraform1.Deferred_RESOURCE_CONFIG_UNKNOWN,
							},
							Change: &terraform1.StackChangeProgress_ResourceInstancePlannedChange{
								Addr: &terraform1.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: "component.deferred",
									ResourceInstanceAddr:  "testing_deferred_resource.resource",
								},
								Actions:      []terraform1.ChangeType{terraform1.ChangeType_CREATE},
								ProviderAddr: "registry.terraform.io/hashicorp/testing",
							},
						},
					},
				},
			},
		},
		{
			Event: &terraform1.PlanStackChanges_Event_Progress{
				Progress: &terraform1.StackChangeProgress{
					Event: &terraform1.StackChangeProgress_ResourceInstanceStatus_{
						ResourceInstanceStatus: &terraform1.StackChangeProgress_ResourceInstanceStatus{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.deferred",
								ResourceInstanceAddr:  "testing_deferred_resource.resource",
							},
							Status:       terraform1.StackChangeProgress_ResourceInstanceStatus_PLANNING,
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				},
			},
		},
		{
			Event: &terraform1.PlanStackChanges_Event_Progress{
				Progress: &terraform1.StackChangeProgress{
					Event: &terraform1.StackChangeProgress_ResourceInstanceStatus_{
						ResourceInstanceStatus: &terraform1.StackChangeProgress_ResourceInstanceStatus{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.deferred",
								ResourceInstanceAddr:  "testing_deferred_resource.resource",
							},
							Status:       terraform1.StackChangeProgress_ResourceInstanceStatus_PLANNED,
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				},
			},
		},
	})
	gotEvents := splitStackOperationEvents(func() []*terraform1.PlanStackChanges_Event {
		var events []*terraform1.PlanStackChanges_Event
		for {
			event, err := resp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			events = append(events, event)
		}
		return events
	}())

	if len(gotEvents.Diagnostics) > 0 {
		for _, evt := range gotEvents.Diagnostics {
			t.Logf("diagnostic: %s", evt.String())
		}
		t.Fatalf("unexpected diagnostics")
	}

	// Now we're going to manually verify the existence of some key events.
	// We're not looking for every event because (a) the exact ordering of
	// events is not guaranteed and (b) we don't want to start failing every
	// time a new event is added.

WantPlannedChange:
	for _, want := range wantEvents.PlannedChanges {
		for _, got := range gotEvents.PlannedChanges {
			if len(cmp.Diff(want, got, protocmp.Transform())) == 0 {
				continue WantPlannedChange
			}
		}
		t.Errorf("missing expected planned change: %v", want)
	}

WantMiscHook:
	for _, want := range wantEvents.MiscHooks {
		for _, got := range gotEvents.MiscHooks {
			if len(cmp.Diff(want, got, protocmp.Transform())) == 0 {
				continue WantMiscHook
			}
		}
		t.Errorf("missing expected event: %v", want)
	}

	if t.Failed() {
		// if the test failed, let's print out all the events we got to help
		// with debugging.
		for _, evt := range gotEvents.MiscHooks {
			t.Logf("        returned event: %s", evt.String())
		}

		for _, evt := range gotEvents.PlannedChanges {
			t.Logf("        returned event: %s", evt.String())
		}
	}

}

// stackOperationEventStreams represents the three different kinds of events
// whose emission is independent from one another and so the relative ordering
// between them is not guaranteed between runs. For easier comparison in
// tests, use splitStackOperationEvents to obtain a value of this type.
//
// Note that even after splitting the streams will not be directly comparable
// for most non-trivial operations, because a typical configuration only
// forces a partial order of operations. Except in carefully-crafted tests
// that are explicitly testing an explicit ordering, it may be better to
// just scan the entire event stream and cherry-pick particular events of
// interest, which will also avoid the need to update every test whenever we
// add something entirely new to the even stream.
type stackOperationEventStreams struct {
	PlannedChanges []*terraform1.PlanStackChanges_Event
	Diagnostics    []*terraform1.PlanStackChanges_Event

	// MiscHooks is the "everything else" category where the detailed begin/end
	// events for individual Terraform Core operations appear.
	MiscHooks []*terraform1.PlanStackChanges_Event
}

func splitStackOperationEvents(all []*terraform1.PlanStackChanges_Event) stackOperationEventStreams {
	ret := stackOperationEventStreams{}
	for _, evt := range all {
		switch evt.Event.(type) {
		case *terraform1.PlanStackChanges_Event_PlannedChange:
			ret.PlannedChanges = append(ret.PlannedChanges, evt)
		case *terraform1.PlanStackChanges_Event_Diagnostic:
			ret.Diagnostics = append(ret.Diagnostics, evt)
		default:
			ret.MiscHooks = append(ret.MiscHooks, evt)
		}
	}
	return ret
}
