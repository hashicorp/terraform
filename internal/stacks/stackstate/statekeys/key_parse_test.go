// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/states"
)

func TestParse(t *testing.T) {
	tests := []struct {
		Input   string
		Want    Key
		WantErr string

		WantUnrecognizedHandling UnrecognizedKeyHandling
	}{
		{
			Input:   "",
			WantErr: `too short to be a valid state key`,
		},
		{
			Input:   "a",
			WantErr: `too short to be a valid state key`,
		},
		{
			Input:   "aa",
			WantErr: `too short to be a valid state key`,
		},
		{
			Input:   "aaa",
			WantErr: `too short to be a valid state key`,
		},
		{
			Input:   "aaa!", // this is a suitable length but contains an invalid character
			WantErr: `invalid key type prefix "aaa!"`,
		},
		{
			Input: "aaaa",
			Want: Unrecognized{
				ApparentKeyType: KeyType("aaaa"),
				remainder:       "",
			},
			WantUnrecognizedHandling: DiscardIfUnrecognized,
		},
		{
			Input: "AAAA",
			Want: Unrecognized{
				ApparentKeyType: KeyType("AAAA"),
				remainder:       "",
			},
			WantUnrecognizedHandling: FailIfUnrecognized,
		},
		{
			Input: "aaaA",
			Want: Unrecognized{
				ApparentKeyType: KeyType("aaaA"),
				remainder:       "",
			},
			WantUnrecognizedHandling: PreserveIfUnrecognized,
		},

		// Resource instance object keys
		{
			Input:   "RSRC",
			WantErr: `resource instance object key has invalid component instance address ""`,
		},
		{
			Input: "RSRCcomponent.foo,aws_instance.bar,cur",
			Want: ResourceInstanceObject{
				ResourceInstance: stackaddrs.AbsResourceInstance{
					Component: stackaddrs.AbsComponentInstance{
						Stack: stackaddrs.RootStackInstance,
						Item: stackaddrs.ComponentInstance{
							Component: stackaddrs.Component{
								Name: "foo",
							},
						},
					},
					Item: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "aws_instance",
								Name: "bar",
							},
						},
					},
				},
				DeposedKey: states.NotDeposed,
			},
			WantUnrecognizedHandling: FailIfUnrecognized,
		},
		{
			// Commas inside quoted instance keys are not treated as
			// delimiters.
			Input: `RSRCcomponent.foo["a,a"],aws_instance.bar["c,c"],cur`,
			Want: ResourceInstanceObject{
				ResourceInstance: stackaddrs.AbsResourceInstance{
					Component: stackaddrs.AbsComponentInstance{
						Stack: stackaddrs.RootStackInstance,
						Item: stackaddrs.ComponentInstance{
							Component: stackaddrs.Component{
								Name: "foo",
							},
							Key: addrs.StringKey("a,a"),
						},
					},
					Item: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "aws_instance",
								Name: "bar",
							},
							Key: addrs.StringKey("c,c"),
						},
					},
				},
				DeposedKey: states.NotDeposed,
			},
			WantUnrecognizedHandling: FailIfUnrecognized,
		},
		{
			// Commas inside quoted instance keys are not treated as
			// delimiters even when there's quote-escaping hazards.
			Input: `RSRCcomponent.foo["a\",a"],aws_instance.bar["c\",c"],cur`,
			Want: ResourceInstanceObject{
				ResourceInstance: stackaddrs.AbsResourceInstance{
					Component: stackaddrs.AbsComponentInstance{
						Stack: stackaddrs.RootStackInstance,
						Item: stackaddrs.ComponentInstance{
							Component: stackaddrs.Component{
								Name: "foo",
							},
							Key: addrs.StringKey(`a",a`),
						},
					},
					Item: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "aws_instance",
								Name: "bar",
							},
							Key: addrs.StringKey(`c",c`),
						},
					},
				},
				DeposedKey: states.NotDeposed,
			},
			WantUnrecognizedHandling: FailIfUnrecognized,
		},
		{
			Input: `RSRCstack.beep["a"].component.foo["b"],module.boop[1].aws_instance.bar[2],cur`,
			Want: ResourceInstanceObject{
				ResourceInstance: stackaddrs.AbsResourceInstance{
					Component: stackaddrs.AbsComponentInstance{
						Stack: stackaddrs.RootStackInstance.Child("beep", addrs.StringKey("a")),
						Item: stackaddrs.ComponentInstance{
							Component: stackaddrs.Component{
								Name: "foo",
							},
							Key: addrs.StringKey("b"),
						},
					},
					Item: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance.Child("boop", addrs.IntKey(1)),
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "aws_instance",
								Name: "bar",
							},
							Key: addrs.IntKey(2),
						},
					},
				},
				DeposedKey: states.NotDeposed,
			},
		},
		{
			Input: "RSRCcomponent.foo,aws_instance.bar,facecafe",
			Want: ResourceInstanceObject{
				ResourceInstance: stackaddrs.AbsResourceInstance{
					Component: stackaddrs.AbsComponentInstance{
						Stack: stackaddrs.RootStackInstance,
						Item: stackaddrs.ComponentInstance{
							Component: stackaddrs.Component{
								Name: "foo",
							},
						},
					},
					Item: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "aws_instance",
								Name: "bar",
							},
						},
					},
				},
				DeposedKey: states.DeposedKey("facecafe"),
			},
		},
		{
			Input:   "RSRCcomponent.foo,aws_instance.bar,beef", // deposed key is invalid because it's not long enough
			WantErr: `resource instance object key has invalid deposed key "beef"`,
		},
		{
			Input:   "RSRCcomponent.foo,aws_instance.bar,tootcafe", // deposed key is invalid because it isn't all hex digits
			WantErr: `resource instance object key has invalid deposed key "tootcafe"`,
		},
		{
			Input:   "RSRCcomponent.foo,aws_instance.bar,FACECAFE", // deposed key is invalid because it uses uppercase hex digits
			WantErr: `resource instance object key has invalid deposed key "FACECAFE"`,
		},
		{
			Input:   "RSRCcomponent.foo,aws_instance.bar,", // last field must either be "cur" or a deposed key
			WantErr: `resource instance object key has invalid deposed key ""`,
		},
		{
			Input:   "RSRCcomponent.foo,aws_instance.bar,cur,",
			WantErr: `unsupported extra field in resource instance object key`,
		},

		// Component instance keys
		{
			Input:   "CMPT",
			WantErr: `component instance key has invalid component instance address ""`,
		},
		{
			Input: "CMPTcomponent.foo",
			Want: ComponentInstance{
				ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{
							Name: "foo",
						},
					},
				},
			},
			WantUnrecognizedHandling: FailIfUnrecognized,
		},
		{
			Input: `CMPTcomponent.foo["baz"]`,
			Want: ComponentInstance{
				ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{
							Name: "foo",
						},
						Key: addrs.StringKey("baz"),
					},
				},
			},
		},
		{
			Input: `CMPTstack.boop.component.foo["baz"]`,
			Want: ComponentInstance{
				ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance.Child("boop", addrs.NoKey),
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{
							Name: "foo",
						},
						Key: addrs.StringKey("baz"),
					},
				},
			},
		},
		{
			Input: `CMPTcomponent.foo["b,b"]`,
			Want: ComponentInstance{
				ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{
							Name: "foo",
						},
						Key: addrs.StringKey(`b,b`),
					},
				},
			},
		},
		{
			Input: `CMPTcomponent.foo["b\",b"]`,
			Want: ComponentInstance{
				ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{
							Name: "foo",
						},
						Key: addrs.StringKey(`b",b`),
					},
				},
			},
		},
		{
			Input:   "CMPTcomponent.foo,",
			WantErr: `unsupported extra field in component instance key`,
		},
	}

	cmpOpts := cmp.AllowUnexported(Unrecognized{})

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			got, err := Parse(test.Input)

			if diff := cmp.Diff(test.Want, got, cmpOpts); diff != "" {
				t.Errorf("wrong result for: %s\n%s", test.Input, diff)
			}

			if test.WantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}

				// Any valid key should round-trip back to what we were given.
				if got != nil {
					gotAsStr := String(got)
					if gotAsStr != test.Input {
						t.Errorf("valid key of type %T did not round-trip\ngot:  %s\nwant: %s", got, gotAsStr, test.Input)
					}
					if test.WantUnrecognizedHandling != UnrecognizedKeyHandling(0) {
						if got, want := got.KeyType().UnrecognizedKeyHandling(), test.WantUnrecognizedHandling; got != want {
							t.Errorf("unexpected UnrecognizedKeyHandling\ngot:  %s\nwant: %s", got, want)
						}
					}
				} else if err == nil {
					t.Error("Parse returned nil Key and nil error")
				}
			} else {
				if err == nil {
					t.Errorf("unexpected success\nwant error: %s", test.WantErr)
				} else {
					if got, want := err.Error(), test.WantErr; got != want {
						t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
					}
				}
			}
		})
	}
}
