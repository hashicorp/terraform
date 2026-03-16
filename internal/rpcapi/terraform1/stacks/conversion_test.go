// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stacks

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

func TestActionTriggerEventForStackChangeProgress(t *testing.T) {
	tests := []struct {
		event   configs.ActionTriggerEvent
		want    StackChangeProgress_ActionTriggerEvent
		wantErr bool
	}{
		{configs.BeforeCreate, StackChangeProgress_BEFORE_CREATE, false},
		{configs.AfterCreate, StackChangeProgress_AFTER_CREATE, false},
		{configs.BeforeUpdate, StackChangeProgress_BEFORE_UPDATE, false},
		{configs.AfterUpdate, StackChangeProgress_AFTER_UPDATE, false},
		{configs.BeforeDestroy, StackChangeProgress_BEFORE_DESTROY, false},
		{configs.AfterDestroy, StackChangeProgress_AFTER_DESTROY, false},
		{configs.Invoke, StackChangeProgress_INVOKE, false},
		{configs.Unknown, StackChangeProgress_INVALID_EVENT, true},
	}

	for _, tt := range tests {
		t.Run(tt.event.String(), func(t *testing.T) {
			got, err := ActionTriggerEventForStackChangeProgress(tt.event)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ActionTriggerEventForStackChangeProgress(%v) error = %v, wantErr %v", tt.event, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ActionTriggerEventForStackChangeProgress(%v) = %v, want %v", tt.event, got, tt.want)
			}
		})
	}
}

func TestActionTriggerEventForPlannedChange(t *testing.T) {
	tests := []struct {
		event   configs.ActionTriggerEvent
		want    PlannedChange_ActionTriggerEvent
		wantErr bool
	}{
		{configs.BeforeCreate, PlannedChange_BEFORE_CREATE, false},
		{configs.AfterCreate, PlannedChange_AFTER_CREATE, false},
		{configs.BeforeUpdate, PlannedChange_BEFORE_UPDATE, false},
		{configs.AfterUpdate, PlannedChange_AFTER_UPDATE, false},
		{configs.BeforeDestroy, PlannedChange_BEFORE_DESTROY, false},
		{configs.AfterDestroy, PlannedChange_AFTER_DESTROY, false},
		{configs.Invoke, PlannedChange_INVOKE, false},
		{configs.Unknown, PlannedChange_INVALID_EVENT, true},
	}

	for _, tt := range tests {
		t.Run(tt.event.String(), func(t *testing.T) {
			got, err := ActionTriggerEventForPlannedChange(tt.event)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ActionTriggerEventForPlannedChange(%v) error = %v, wantErr %v", tt.event, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ActionTriggerEventForPlannedChange(%v) = %v, want %v", tt.event, got, tt.want)
			}
		})
	}
}

func TestNewActionInvocationInStackAddr(t *testing.T) {
	tests := []struct {
		name string
		addr stackaddrs.AbsActionInvocationInstance
		want *ActionInvocationInstanceInStackAddr
	}{
		{
			name: "simple action in root component",
			addr: stackaddrs.AbsActionInvocationInstance{
				Component: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{Name: "foo"},
					},
				},
				Item: addrs.AbsActionInstance{
					Module: addrs.RootModuleInstance,
					Action: addrs.ActionInstance{
						Action: addrs.Action{
							Type: "example",
							Name: "test",
						},
						Key: addrs.NoKey,
					},
				},
			},
			want: &ActionInvocationInstanceInStackAddr{
				ComponentInstanceAddr:        "component.foo",
				ActionInvocationInstanceAddr: "action.example.test",
			},
		},
		{
			name: "action with count index",
			addr: stackaddrs.AbsActionInvocationInstance{
				Component: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{Name: "bar"},
					},
				},
				Item: addrs.AbsActionInstance{
					Module: addrs.RootModuleInstance,
					Action: addrs.ActionInstance{
						Action: addrs.Action{
							Type: "webhook",
							Name: "notify",
						},
						Key: addrs.IntKey(2),
					},
				},
			},
			want: &ActionInvocationInstanceInStackAddr{
				ComponentInstanceAddr:        "component.bar",
				ActionInvocationInstanceAddr: "action.webhook.notify[2]",
			},
		},
		{
			name: "action with for_each key",
			addr: stackaddrs.AbsActionInvocationInstance{
				Component: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{Name: "baz"},
						Key:       addrs.StringKey("prod"),
					},
				},
				Item: addrs.AbsActionInstance{
					Module: addrs.RootModuleInstance,
					Action: addrs.ActionInstance{
						Action: addrs.Action{
							Type: "slack",
							Name: "alert",
						},
						Key: addrs.StringKey("critical"),
					},
				},
			},
			want: &ActionInvocationInstanceInStackAddr{
				ComponentInstanceAddr:        `component.baz["prod"]`,
				ActionInvocationInstanceAddr: `action.slack.alert["critical"]`,
			},
		},
		{
			name: "action in child module",
			addr: stackaddrs.AbsActionInvocationInstance{
				Component: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{Name: "network"},
					},
				},
				Item: addrs.AbsActionInstance{
					Module: addrs.RootModuleInstance.Child("vpc", addrs.NoKey),
					Action: addrs.ActionInstance{
						Action: addrs.Action{
							Type: "aws_lambda",
							Name: "cleanup",
						},
						Key: addrs.NoKey,
					},
				},
			},
			want: &ActionInvocationInstanceInStackAddr{
				ComponentInstanceAddr:        "component.network",
				ActionInvocationInstanceAddr: "module.vpc.action.aws_lambda.cleanup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewActionInvocationInStackAddr(tt.addr)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("NewActionInvocationInStackAddr() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewActionInvocationInStackAddr_RoundTrip(t *testing.T) {
	// Test that the string representation is stable and meaningful
	addr := stackaddrs.AbsActionInvocationInstance{
		Component: stackaddrs.AbsComponentInstance{
			Stack: stackaddrs.RootStackInstance,
			Item: stackaddrs.ComponentInstance{
				Component: stackaddrs.Component{Name: "mycomp"},
			},
		},
		Item: addrs.AbsActionInstance{
			Module: addrs.RootModuleInstance,
			Action: addrs.ActionInstance{
				Action: addrs.Action{
					Type: "http",
					Name: "webhook",
				},
				Key: addrs.NoKey,
			},
		},
	}

	proto := NewActionInvocationInStackAddr(addr)

	// Verify the strings match what we expect
	if proto.ComponentInstanceAddr != "component.mycomp" {
		t.Errorf("unexpected component address: %q", proto.ComponentInstanceAddr)
	}
	if proto.ActionInvocationInstanceAddr != "action.http.webhook" {
		t.Errorf("unexpected action address: %q", proto.ActionInvocationInstanceAddr)
	}
}
