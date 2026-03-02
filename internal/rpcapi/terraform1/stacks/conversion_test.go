// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stacks

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

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
