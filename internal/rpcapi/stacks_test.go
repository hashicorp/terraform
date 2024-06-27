// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/zclconf/go-cty/cty"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/states"
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
				"unused":              {Optional: false},
				"unused_with_default": {Optional: true},
				"sensitive":           {Sensitive: true},
				"ephemeral":           {Ephemeral: true},
			},
			OutputValues: map[string]*terraform1.FindStackConfigurationComponents_OutputValue{
				"normal":    {},
				"sensitive": {Sensitive: true},
				"ephemeral": {Ephemeral: true},
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
						mustMarshalAnyPb(&tfstackdata1.PlanTimestamp{
							PlanTimestamp: time.Now().UTC().Format(time.RFC3339),
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
							Applyable: false,
						}),
					},
					Descriptions: []*terraform1.PlannedChange_ChangeDescription{
						{
							Description: &terraform1.PlannedChange_ChangeDescription_PlanApplyable{
								PlanApplyable: false,
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

func TestStackChangeProgress(t *testing.T) {
	tcs := map[string]struct {
		source      string
		store       *stacks_testing_provider.ResourceStore
		state       []stackstate.AppliedChange
		inputs      map[string]cty.Value
		want        []*terraform1.StackChangeProgress
		diagnostics []*terraform1.Diagnostic
	}{
		"deferred_changes": {
			source: "git::https://example.com/bar.git",
			want: []*terraform1.StackChangeProgress{
				{
					Event: &terraform1.StackChangeProgress_ComponentInstanceChanges_{
						ComponentInstanceChanges: &terraform1.StackChangeProgress_ComponentInstanceChanges{
							Addr: &terraform1.ComponentInstanceInStackAddr{
								ComponentAddr:         "component.deferred",
								ComponentInstanceAddr: "component.deferred",
							},
							Total: 1,
							Defer: 1,
						},
					},
				},
				{
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
				{
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
				{
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
		"moved": {
			source: "git::https://example.com/moved.git",
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("before", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("before"),
					"value": cty.NullVal(cty.String),
				})).
				Build(),
			state: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent(t, "component.self"),
					ComponentInstanceAddr: mustAbsComponentInstance(t, "component.self"),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject(t, "component.self.testing_resource.before"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "before",
							"value": nil,
						}),
						Status: states.ObjectReady,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
			want: []*terraform1.StackChangeProgress{
				{
					Event: &terraform1.StackChangeProgress_ResourceInstancePlannedChange_{
						ResourceInstancePlannedChange: &terraform1.StackChangeProgress_ResourceInstancePlannedChange{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.self",
								ResourceInstanceAddr:  "testing_resource.after",
							},
							Actions: []terraform1.ChangeType{
								terraform1.ChangeType_NOOP,
							},
							Moved: &terraform1.StackChangeProgress_ResourceInstancePlannedChange_Moved{
								PrevAddr: &terraform1.ResourceInstanceInStackAddr{
									ComponentInstanceAddr: "component.self",
									ResourceInstanceAddr:  "testing_resource.before",
								},
							},
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				},
				{
					Event: &terraform1.StackChangeProgress_ComponentInstanceChanges_{
						ComponentInstanceChanges: &terraform1.StackChangeProgress_ComponentInstanceChanges{
							Addr: &terraform1.ComponentInstanceInStackAddr{
								ComponentAddr:         "component.self",
								ComponentInstanceAddr: "component.self",
							},
							Total: 1,
							Move:  1,
						},
					},
				},
			},
		},
		"import": {
			source: "git::https://example.com/import.git",
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("self", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("self"),
					"value": cty.NullVal(cty.String),
				})).
				Build(),
			inputs: map[string]cty.Value{
				"unknown": cty.UnknownVal(cty.String),
			},
			want: []*terraform1.StackChangeProgress{
				{

					Event: &terraform1.StackChangeProgress_ComponentInstanceChanges_{
						ComponentInstanceChanges: &terraform1.StackChangeProgress_ComponentInstanceChanges{
							Addr: &terraform1.ComponentInstanceInStackAddr{
								ComponentAddr:         "component.unknown",
								ComponentInstanceAddr: "component.unknown",
							},
							Total: 1,
							Defer: 1,
						},
					},
				},
				{
					Event: &terraform1.StackChangeProgress_DeferredResourceInstancePlannedChange_{
						DeferredResourceInstancePlannedChange: &terraform1.StackChangeProgress_DeferredResourceInstancePlannedChange{
							Deferred: &terraform1.Deferred{
								Reason: terraform1.Deferred_RESOURCE_CONFIG_UNKNOWN,
							},
							Change: &terraform1.StackChangeProgress_ResourceInstancePlannedChange{
								Addr: &terraform1.ResourceInstanceObjectInStackAddr{
									ComponentInstanceAddr: "component.unknown",
									ResourceInstanceAddr:  "testing_resource.resource",
								},
								Actions: []terraform1.ChangeType{terraform1.ChangeType_CREATE},
								Imported: &terraform1.StackChangeProgress_ResourceInstancePlannedChange_Imported{
									Unknown: true,
								},
								ProviderAddr: "registry.terraform.io/hashicorp/testing",
							},
						},
					},
				},
				{
					Event: &terraform1.StackChangeProgress_ResourceInstanceStatus_{
						ResourceInstanceStatus: &terraform1.StackChangeProgress_ResourceInstanceStatus{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.unknown",
								ResourceInstanceAddr:  "testing_resource.resource",
							},
							Status:       terraform1.StackChangeProgress_ResourceInstanceStatus_PLANNING,
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				},
				{
					Event: &terraform1.StackChangeProgress_ResourceInstanceStatus_{
						ResourceInstanceStatus: &terraform1.StackChangeProgress_ResourceInstanceStatus{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.unknown",
								ResourceInstanceAddr:  "testing_resource.resource",
							},
							Status:       terraform1.StackChangeProgress_ResourceInstanceStatus_PLANNED,
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				}, {
					Event: &terraform1.StackChangeProgress_ComponentInstanceChanges_{
						ComponentInstanceChanges: &terraform1.StackChangeProgress_ComponentInstanceChanges{
							Addr: &terraform1.ComponentInstanceInStackAddr{
								ComponentAddr:         "component.self",
								ComponentInstanceAddr: "component.self",
							},
							Total:  1,
							Import: 1,
						},
					},
				},
				{
					Event: &terraform1.StackChangeProgress_ResourceInstancePlannedChange_{
						ResourceInstancePlannedChange: &terraform1.StackChangeProgress_ResourceInstancePlannedChange{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.self",
								ResourceInstanceAddr:  "testing_resource.resource",
							},
							Actions: []terraform1.ChangeType{terraform1.ChangeType_NOOP},
							Imported: &terraform1.StackChangeProgress_ResourceInstancePlannedChange_Imported{
								ImportId: "self",
							},
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				},
				{
					Event: &terraform1.StackChangeProgress_ResourceInstanceStatus_{
						ResourceInstanceStatus: &terraform1.StackChangeProgress_ResourceInstanceStatus{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.self",
								ResourceInstanceAddr:  "testing_resource.resource",
							},
							Status:       terraform1.StackChangeProgress_ResourceInstanceStatus_PLANNING,
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				},
				{
					Event: &terraform1.StackChangeProgress_ResourceInstanceStatus_{
						ResourceInstanceStatus: &terraform1.StackChangeProgress_ResourceInstanceStatus{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.self",
								ResourceInstanceAddr:  "testing_resource.resource",
							},
							Status:       terraform1.StackChangeProgress_ResourceInstanceStatus_PLANNED,
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				},
			},
		},
		"removed": {
			source: "git::https://example.com/removed.git",
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("resource", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("resource"),
					"value": cty.NullVal(cty.String),
				})).
				Build(),
			state: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent(t, "component.self"),
					ComponentInstanceAddr: mustAbsComponentInstance(t, "component.self"),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject(t, "component.self.testing_resource.resource"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "resource",
							"value": nil,
						}),
						Status: states.ObjectReady,
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
			want: []*terraform1.StackChangeProgress{
				{
					Event: &terraform1.StackChangeProgress_ResourceInstancePlannedChange_{
						ResourceInstancePlannedChange: &terraform1.StackChangeProgress_ResourceInstancePlannedChange{
							Addr: &terraform1.ResourceInstanceObjectInStackAddr{
								ComponentInstanceAddr: "component.self",
								ResourceInstanceAddr:  "testing_resource.resource",
							},
							Actions: []terraform1.ChangeType{
								terraform1.ChangeType_FORGET,
							},
							ProviderAddr: "registry.terraform.io/hashicorp/testing",
						},
					},
				},
				{
					Event: &terraform1.StackChangeProgress_ComponentInstanceChanges_{
						ComponentInstanceChanges: &terraform1.StackChangeProgress_ComponentInstanceChanges{
							Addr: &terraform1.ComponentInstanceInStackAddr{
								ComponentAddr:         "component.self",
								ComponentInstanceAddr: "component.self",
							},
							Total:  1,
							Forget: 1,
						},
					},
				},
			},
			diagnostics: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_WARNING,
					Summary:  "Some objects will no longer be managed by Terraform",
					Detail:   "If you apply this plan, Terraform will discard its tracking information for the following objects, but it will not delete them:\n - testing_resource.resource\n\nAfter applying this plan, Terraform will no longer manage these objects. You will need to import them into Terraform to manage them again.",
				},
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			handles := newHandleTable()
			stacksServer := newStacksServer(newStopper(), handles, &serviceOpts{})

			// For this test, we do actually want to use a "real" provider. We'll
			// use the providerCacheOverride to side-load the testing provider.
			stacksServer.providerCacheOverride = make(map[addrs.Provider]providers.Factory)
			stacksServer.providerCacheOverride[addrs.NewDefaultProvider("testing")] = func() (providers.Interface, error) {
				return stacks_testing_provider.NewProviderWithData(tc.store), nil
			}
			lock := depsfile.NewLocks()
			lock.SetProvider(
				addrs.NewDefaultProvider("testing"),
				providerreqs.MustParseVersion("0.0.0"),
				providerreqs.MustParseVersionConstraints("=0.0.0"),
				providerreqs.PreferredHashes([]providerreqs.Hash{}),
			)
			stacksServer.providerDependencyLockOverride = lock

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
					Source: tc.source,
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
				PreviousState:     appliedChangeToRawState(t, tc.state),
				InputValues: func() map[string]*terraform1.DynamicValueWithSource {
					values := make(map[string]*terraform1.DynamicValueWithSource)
					for name, value := range tc.inputs {
						values[name] = &terraform1.DynamicValueWithSource{
							Value: &terraform1.DynamicValue{
								Msgpack: mustMsgpack(t, value, value.Type()),
							},
							SourceRange: &terraform1.SourceRange{
								Start: &terraform1.SourcePos{},
								End:   &terraform1.SourcePos{},
							},
						}
					}
					return values
				}(),
			})
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			wantEvents := splitStackOperationEvents(func() []*terraform1.PlanStackChanges_Event {
				events := make([]*terraform1.PlanStackChanges_Event, 0, len(tc.want))
				for _, want := range tc.want {
					events = append(events, &terraform1.PlanStackChanges_Event{
						Event: &terraform1.PlanStackChanges_Event_Progress{
							Progress: want,
						},
					})
				}
				return events
			}())

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

			// First, validate the diagnostics. Most of the tests are either
			// expecting a specific single diagnostic so we do actually check
			// everything.

			diagIx := 0
			for ; diagIx < len(tc.diagnostics); diagIx++ {
				if diagIx >= len(gotEvents.Diagnostics) {
					// Then we have more expected diagnostics than we got.
					t.Errorf("missing expected diagnostic: %v", tc.diagnostics[diagIx])
					continue
				}
				diag := gotEvents.Diagnostics[diagIx].Event.(*terraform1.PlanStackChanges_Event_Diagnostic).Diagnostic
				if diff := cmp.Diff(tc.diagnostics[diagIx], diag, protocmp.Transform()); diff != "" {
					// Then we have a diagnostic that doesn't match what we
					// expected.
					t.Errorf("wrong diagnostic\n%s", diff)
				}
			}
			for ; diagIx < len(gotEvents.Diagnostics); diagIx++ {
				// Then we have more diagnostics than we expected.
				t.Errorf("unexpected diagnostic: %v", gotEvents.Diagnostics[diagIx])
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
		})
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

func mustMsgpack(t *testing.T, v cty.Value, ty cty.Type) []byte {
	t.Helper()

	ret, err := ctymsgpack.Marshal(v, ty)
	if err != nil {
		t.Fatalf("error marshalling %#v: %s", v, err)
	}

	return ret
}
