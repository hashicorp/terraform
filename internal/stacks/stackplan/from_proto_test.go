// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans/planproto"
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

func TestAddRawActionInvocation(t *testing.T) {
	loader := NewLoader()

	// Add component instance first
	componentRaw := mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
		ComponentInstanceAddr: "stack.root.component.foo",
		PlannedAction:         planproto.Action_NOOP,
		Mode:                  planproto.Mode_NORMAL,
		PlanApplyable:         true,
		PlanComplete:          true,
		PlanTimestamp:         "2023-01-01T00:00:00Z",
	})
	if err := loader.AddRaw(componentRaw); err != nil {
		t.Fatalf("AddRaw() component error = %v", err)
	}

	// Add action invocation
	actionRaw := mustMarshalAnyPb(&tfstackdata1.PlanActionInvocationPlanned{
		ComponentInstanceAddr: "stack.root.component.foo",
		ActionInvocationAddr:  "action.example.test",
		ProviderConfigAddr:    "provider[\"registry.terraform.io/hashicorp/testing\"]",
		Invocation: &planproto.ActionInvocationInstance{
			Addr:     "action.example.test",
			Provider: "provider[\"registry.terraform.io/hashicorp/testing\"]",
			ActionTrigger: &planproto.ActionInvocationInstance_InvokeActionTrigger{
				InvokeActionTrigger: &planproto.InvokeActionTrigger{},
			},
		},
	})
	if err := loader.AddRaw(actionRaw); err != nil {
		t.Fatalf("AddRaw() action error = %v", err)
	}

	plan := loader.ret

	// Verify the component was created
	componentAddr, err := stackaddrs.ParseAbsComponentInstanceStr("stack.root.component.foo")
	if err != nil {
		t.Fatalf("failed to parse component address: %v", err)
	}

	component, componentFound := plan.Root.GetOk(componentAddr)
	if !componentFound {
		t.Fatalf("expected component %s to be present in plan", componentAddr)
	}

	// Verify the action invocation was added to the component
	if len(component.ActionInvocations.Elems) != 1 {
		t.Fatalf("expected 1 action invocation, got %d", len(component.ActionInvocations.Elems))
	}

	// Check that the action invocation has the correct address
	if component.ActionInvocations.Len() == 0 {
		t.Fatal("expected action invocations to be non-empty")
	}

	// Iterate over the action invocations to find our test action
	expectedActionAddr := "action.example.test"
	actionFound := false
	for _, elem := range component.ActionInvocations.Elems {
		actionAddr := elem.Key
		if actionAddr.String() == expectedActionAddr {
			actionFound = true
			break
		}
	}

	if !actionFound {
		t.Errorf("expected to find action address %s in component action invocations", expectedActionAddr)
	}
}
func TestAddRawActionInvocation_InvalidAddr(t *testing.T) {
	loader := NewLoader()

	// Valid component
	loader.AddRaw(mustMarshalAnyPb(&tfstackdata1.PlanComponentInstance{
		ComponentInstanceAddr: "stack.root.component.foo",
	}))

	// Invalid action invocation (empty address)
	loader.AddRaw(mustMarshalAnyPb(&tfstackdata1.PlanActionInvocationPlanned{
		ComponentInstanceAddr: "stack.root.component.foo",
		ActionInvocationAddr:  "",
	}))

	componentAddr, err := stackaddrs.ParseAbsComponentInstanceStr("stack.root.component.foo")
	if err != nil {
		t.Fatalf("failed to parse component address: %v", err)
	}
	component, ok := loader.ret.Root.GetOk(componentAddr)
	if !ok {
		t.Fatalf("component not found")
	}
	if component.ActionInvocations.Len() != 0 {
		t.Errorf("expected no action invocations for invalid address")
	}
}
