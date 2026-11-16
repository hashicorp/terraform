// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
)

func TestAddRaw(t *testing.T) {
	tests := map[string]struct {
		Raw  []*anypb.Any
		Want *Plan
	}{
		"empty": {
			Raw: nil,
			Want: &Plan{
				Root:            newStackInstance(stackaddrs.RootStackInstance),
				PrevRunStateRaw: make(map[string]*anypb.Any),
				RootInputValues: make(map[stackaddrs.InputVariable]cty.Value),
			},
		},
		"sensitive input value": {
			Raw: []*anypb.Any{
				mustMarshalAnyPb(&tfstackdata1.PlanRootInputValue{
					Name: "foo",
					Value: &tfstackdata1.DynamicValue{
						Value: &planproto.DynamicValue{
							Msgpack: []byte("\x92\xc4\b\"string\"\xa4boop"),
						},
						SensitivePaths: []*planproto.Path{
							{
								Steps: make([]*planproto.Path_Step, 0), // no steps as it is the root value
							},
						},
					},
					RequiredOnApply: false,
				}),
			},
			Want: &Plan{
				Root:            newStackInstance(stackaddrs.RootStackInstance),
				PrevRunStateRaw: make(map[string]*anypb.Any),
				RootInputValues: map[stackaddrs.InputVariable]cty.Value{
					stackaddrs.InputVariable{Name: "foo"}: cty.StringVal("boop").Mark(marks.Sensitive),
				},
			},
		},
		"input value": {
			Raw: []*anypb.Any{
				mustMarshalAnyPb(&tfstackdata1.PlanRootInputValue{
					Name: "foo",
					Value: &tfstackdata1.DynamicValue{
						Value: &planproto.DynamicValue{
							Msgpack: []byte("\x92\xc4\b\"string\"\xa4boop"),
						},
					},
					RequiredOnApply: false,
				}),
			},
			Want: &Plan{
				Root:            newStackInstance(stackaddrs.RootStackInstance),
				PrevRunStateRaw: make(map[string]*anypb.Any),
				RootInputValues: map[stackaddrs.InputVariable]cty.Value{
					stackaddrs.InputVariable{Name: "foo"}: cty.StringVal("boop"),
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			loader := NewLoader()
			for _, raw := range test.Raw {
				if err := loader.AddRaw(raw); err != nil {
					t.Errorf("AddRaw() error = %v", err)
				}
			}

			if t.Failed() {
				return
			}

			opts := cmp.Options{
				ctydebug.CmpOptions,
				collections.CmpOptions,
			}
			if diff := cmp.Diff(test.Want, loader.ret, opts...); diff != "" {
				t.Errorf("AddRaw() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddRaw_ActionInvocations(t *testing.T) {
	provider := addrs.MustParseProviderSourceString("example.com/test/actions")
	providerConfig := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: provider,
	}
	action := &plans.ActionInvocationInstanceSrc{
		Addr: addrs.RootModuleInstance.ActionInstance("webhook", "notify", addrs.NoKey),
		ActionTrigger: &plans.ResourceActionTrigger{
			TriggeringResourceAddr:  addrs.RootModuleInstance.ResourceInstance(addrs.ManagedResourceMode, "example_resource", "main", addrs.NoKey),
			ActionTriggerEvent:      configs.AfterCreate,
			ActionTriggerBlockIndex: 0,
			ActionsListIndex:        0,
		},
		ProviderAddr: providerConfig,
	}
	rawAction, err := planfile.ActionInvocationToProto(action)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	err = loader.AddRaw(mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
		ComponentInstanceAddr: "component.web",
		PlannedAction:         planproto.Action_NOOP,
		Mode:                  planproto.Mode_NORMAL,
		PlanTimestamp:         "2017-03-27T10:00:00-08:00",
	}))
	if err != nil {
		t.Fatalf("adding component: %v", err)
	}
	err = loader.AddRaw(mustMarshalAnyPb(&tfstackdata1.PlanActionInvocationPlanned{
		ComponentInstanceAddr: "component.web",
		ActionInvocationAddr:  action.Addr.String(),
		ProviderConfigAddr:    provider.String(),
		Invocation:            rawAction,
	}))
	if err != nil {
		t.Fatalf("adding planned action invocation: %v", err)
	}
	err = loader.AddRaw(mustMarshalAnyPb(&tfstackdata1.PlanDeferredActionInvocation{
		Deferred: &planproto.Deferred{
			Reason: planproto.DeferredReason_DEFERRED_PREREQ,
		},
		Invocation: &tfstackdata1.PlanActionInvocationPlanned{
			ComponentInstanceAddr: "component.web",
			ActionInvocationAddr:  action.Addr.String(),
			ProviderConfigAddr:    provider.String(),
			Invocation:            rawAction,
		},
	}))
	if err != nil {
		t.Fatalf("adding deferred action invocation: %v", err)
	}

	componentAddr, diags := stackaddrs.ParseAbsComponentInstanceStr("component.web")
	if diags.HasErrors() {
		t.Fatalf("parsing component address: %s", diags.Err())
	}
	component := loader.ret.GetComponent(componentAddr)
	if component == nil {
		t.Fatal("expected component to be loaded")
	}

	if len(component.ActionInvocations) != 1 {
		t.Fatalf("expected 1 planned action invocation, got %d", len(component.ActionInvocations))
	}
	if diff := cmp.Diff(action, component.ActionInvocations[0], ctydebug.CmpOptions); diff != "" {
		t.Fatalf("wrong planned action invocation (-want +got):\n%s", diff)
	}
	if len(component.DeferredActionInvocations) != 1 {
		t.Fatalf("expected 1 deferred action invocation, got %d", len(component.DeferredActionInvocations))
	}
	if diff := cmp.Diff(&plans.DeferredActionInvocationSrc{
		DeferredReason:              providers.DeferredReasonDeferredPrereq,
		ActionInvocationInstanceSrc: action,
	}, component.DeferredActionInvocations[0], ctydebug.CmpOptions); diff != "" {
		t.Fatalf("wrong deferred action invocation (-want +got):\n%s", diff)
	}

	modulesPlan, err := component.ForModulesRuntime()
	if err != nil {
		t.Fatalf("ForModulesRuntime: %v", err)
	}
	if len(modulesPlan.Changes.ActionInvocations) != 1 {
		t.Fatalf("expected 1 planned action invocation in modules runtime plan, got %d", len(modulesPlan.Changes.ActionInvocations))
	}
	if diff := cmp.Diff(action, modulesPlan.Changes.ActionInvocations[0], ctydebug.CmpOptions); diff != "" {
		t.Fatalf("wrong modules runtime action invocation (-want +got):\n%s", diff)
	}
	if len(modulesPlan.DeferredActionInvocations) != 1 {
		t.Fatalf("expected 1 deferred action invocation in modules runtime plan, got %d", len(modulesPlan.DeferredActionInvocations))
	}
	if diff := cmp.Diff(&plans.DeferredActionInvocationSrc{
		DeferredReason:              providers.DeferredReasonDeferredPrereq,
		ActionInvocationInstanceSrc: action,
	}, modulesPlan.DeferredActionInvocations[0], ctydebug.CmpOptions); diff != "" {
		t.Fatalf("wrong modules runtime deferred action invocation (-want +got):\n%s", diff)
	}

	requiredProviders := component.RequiredProviderInstances()
	if !requiredProviders.Has(addrs.RootProviderConfig{Provider: provider}) {
		t.Fatalf("expected action provider %s to be required", provider)
	}
}
