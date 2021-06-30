package addrs

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParseMoveEndpoint(t *testing.T) {
	tests := []struct {
		Input   string
		WantRel AbsMoveable // funny intermediate subset of AbsMovable
		WantErr string
	}{
		{
			`foo.bar`,
			AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: NoKey,
				},
			},
			``,
		},
		{
			`foo.bar[0]`,
			AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: IntKey(0),
				},
			},
			``,
		},
		{
			`foo.bar["a"]`,
			AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: StringKey("a"),
				},
			},
			``,
		},
		{
			`module.boop.foo.bar`,
			AbsResourceInstance{
				Module: ModuleInstance{
					ModuleInstanceStep{Name: "boop"},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: NoKey,
				},
			},
			``,
		},
		{
			`module.boop.foo.bar[0]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					ModuleInstanceStep{Name: "boop"},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: IntKey(0),
				},
			},
			``,
		},
		{
			`module.boop.foo.bar["a"]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					ModuleInstanceStep{Name: "boop"},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: StringKey("a"),
				},
			},
			``,
		},
		{
			`data.foo.bar`,
			AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: NoKey,
				},
			},
			``,
		},
		{
			`data.foo.bar[0]`,
			AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: IntKey(0),
				},
			},
			``,
		},
		{
			`data.foo.bar["a"]`,
			AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: StringKey("a"),
				},
			},
			``,
		},
		{
			`module.boop.data.foo.bar`,
			AbsResourceInstance{
				Module: ModuleInstance{
					ModuleInstanceStep{Name: "boop"},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: NoKey,
				},
			},
			``,
		},
		{
			`module.boop.data.foo.bar[0]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					ModuleInstanceStep{Name: "boop"},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: IntKey(0),
				},
			},
			``,
		},
		{
			`module.boop.data.foo.bar["a"]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					ModuleInstanceStep{Name: "boop"},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: DataResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: StringKey("a"),
				},
			},
			``,
		},
		{
			`module.foo`,
			ModuleInstance{
				ModuleInstanceStep{Name: "foo"},
			},
			``,
		},
		{
			`module.foo[0]`,
			ModuleInstance{
				ModuleInstanceStep{Name: "foo", InstanceKey: IntKey(0)},
			},
			``,
		},
		{
			`module.foo["a"]`,
			ModuleInstance{
				ModuleInstanceStep{Name: "foo", InstanceKey: StringKey("a")},
			},
			``,
		},
		{
			`module.foo.module.bar`,
			ModuleInstance{
				ModuleInstanceStep{Name: "foo"},
				ModuleInstanceStep{Name: "bar"},
			},
			``,
		},
		{
			`module.foo[1].module.bar`,
			ModuleInstance{
				ModuleInstanceStep{Name: "foo", InstanceKey: IntKey(1)},
				ModuleInstanceStep{Name: "bar"},
			},
			``,
		},
		{
			`module.foo.module.bar[1]`,
			ModuleInstance{
				ModuleInstanceStep{Name: "foo"},
				ModuleInstanceStep{Name: "bar", InstanceKey: IntKey(1)},
			},
			``,
		},
		{
			`module.foo[0].module.bar[1]`,
			ModuleInstance{
				ModuleInstanceStep{Name: "foo", InstanceKey: IntKey(0)},
				ModuleInstanceStep{Name: "bar", InstanceKey: IntKey(1)},
			},
			``,
		},
		{
			`module`,
			nil,
			`Invalid address operator: Prefix "module." must be followed by a module name.`,
		},
		{
			`module[0]`,
			nil,
			`Invalid address operator: Prefix "module." must be followed by a module name.`,
		},
		{
			`module.foo.data`,
			nil,
			`Invalid address: Resource specification must include a resource type and name.`,
		},
		{
			`module.foo.data.bar`,
			nil,
			`Invalid address: Resource specification must include a resource type and name.`,
		},
		{
			`module.foo.data[0]`,
			nil,
			`Invalid address: Resource specification must include a resource type and name.`,
		},
		{
			`module.foo.data.bar[0]`,
			nil,
			`Invalid address: A resource name is required.`,
		},
		{
			`module.foo.bar`,
			nil,
			`Invalid address: Resource specification must include a resource type and name.`,
		},
		{
			`module.foo.bar[0]`,
			nil,
			`Invalid address: A resource name is required.`,
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.InitialPos)
			if hclDiags.HasErrors() {
				// We're not trying to test the HCL parser here, so any
				// failures at this point are likely to be bugs in the
				// test case itself.
				t.Fatalf("syntax error: %s", hclDiags.Error())
			}

			moveEp, diags := ParseMoveEndpoint(traversal)

			switch {
			case test.WantErr != "":
				if !diags.HasErrors() {
					t.Fatalf("unexpected success\nwant error: %s", test.WantErr)
				}
				gotErr := diags.Err().Error()
				if gotErr != test.WantErr {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", gotErr, test.WantErr)
				}
			default:
				if diags.HasErrors() {
					t.Fatalf("unexpected error: %s", diags.Err().Error())
				}
				if diff := cmp.Diff(test.WantRel, moveEp.relSubject); diff != "" {
					t.Errorf("wrong result\n%s", diff)
				}
			}
		})
	}
}

func TestUnifyMoveEndpoints(t *testing.T) {
	tests := []struct {
		InputFrom, InputTo string
		Module             ModuleInstance
		WantFrom, WantTo   AbsMoveable
	}{
		{
			InputFrom: `foo.bar`,
			InputTo:   `foo.baz`,
			Module:    RootModuleInstance,
			WantFrom: AbsResource{
				Module: RootModuleInstance,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "bar",
				},
			},
			WantTo: AbsResource{
				Module: RootModuleInstance,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "baz",
				},
			},
		},
		{
			InputFrom: `foo.bar`,
			InputTo:   `foo.baz`,
			Module:    RootModuleInstance.Child("a", NoKey),
			WantFrom: AbsResource{
				Module: RootModuleInstance.Child("a", NoKey),
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "bar",
				},
			},
			WantTo: AbsResource{
				Module: RootModuleInstance.Child("a", NoKey),
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "baz",
				},
			},
		},
		{
			InputFrom: `foo.bar`,
			InputTo:   `module.b[0].foo.baz`,
			Module:    RootModuleInstance.Child("a", NoKey),
			WantFrom: AbsResource{
				Module: RootModuleInstance.Child("a", NoKey),
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "bar",
				},
			},
			WantTo: AbsResource{
				Module: RootModuleInstance.Child("a", NoKey).Child("b", IntKey(0)),
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "baz",
				},
			},
		},
		{
			InputFrom: `foo.bar`,
			InputTo:   `foo.bar["thing"]`,
			Module:    RootModuleInstance,
			WantFrom: AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
				},
			},
			WantTo: AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: StringKey("thing"),
				},
			},
		},
		{
			InputFrom: `foo.bar["thing"]`,
			InputTo:   `foo.bar`,
			Module:    RootModuleInstance,
			WantFrom: AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: StringKey("thing"),
				},
			},
			WantTo: AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
				},
			},
		},
		{
			InputFrom: `foo.bar["a"]`,
			InputTo:   `foo.bar["b"]`,
			Module:    RootModuleInstance,
			WantFrom: AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: StringKey("a"),
				},
			},
			WantTo: AbsResourceInstance{
				Module: RootModuleInstance,
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "foo",
						Name: "bar",
					},
					Key: StringKey("b"),
				},
			},
		},
		{
			InputFrom: `module.foo`,
			InputTo:   `module.bar`,
			Module:    RootModuleInstance,
			WantFrom: AbsModuleCall{
				Module: RootModuleInstance,
				Call:   ModuleCall{Name: "foo"},
			},
			WantTo: AbsModuleCall{
				Module: RootModuleInstance,
				Call:   ModuleCall{Name: "bar"},
			},
		},
		{
			InputFrom: `module.foo`,
			InputTo:   `module.bar.module.baz`,
			Module:    RootModuleInstance,
			WantFrom: AbsModuleCall{
				Module: RootModuleInstance,
				Call:   ModuleCall{Name: "foo"},
			},
			WantTo: AbsModuleCall{
				Module: RootModuleInstance.Child("bar", NoKey),
				Call:   ModuleCall{Name: "baz"},
			},
		},
		{
			InputFrom: `module.foo`,
			InputTo:   `module.bar.module.baz`,
			Module:    RootModuleInstance.Child("bloop", StringKey("hi")),
			WantFrom: AbsModuleCall{
				Module: RootModuleInstance.Child("bloop", StringKey("hi")),
				Call:   ModuleCall{Name: "foo"},
			},
			WantTo: AbsModuleCall{
				Module: RootModuleInstance.Child("bloop", StringKey("hi")).Child("bar", NoKey),
				Call:   ModuleCall{Name: "baz"},
			},
		},
		{
			InputFrom: `module.foo[0]`,
			InputTo:   `module.foo["a"]`,
			Module:    RootModuleInstance,
			WantFrom:  RootModuleInstance.Child("foo", IntKey(0)),
			WantTo:    RootModuleInstance.Child("foo", StringKey("a")),
		},
		{
			InputFrom: `module.foo`,
			InputTo:   `module.foo["a"]`,
			Module:    RootModuleInstance,
			WantFrom:  RootModuleInstance.Child("foo", NoKey),
			WantTo:    RootModuleInstance.Child("foo", StringKey("a")),
		},
		{
			InputFrom: `module.foo[0]`,
			InputTo:   `module.foo`,
			Module:    RootModuleInstance,
			WantFrom:  RootModuleInstance.Child("foo", IntKey(0)),
			WantTo:    RootModuleInstance.Child("foo", NoKey),
		},
		{
			InputFrom: `module.foo[0]`,
			InputTo:   `module.foo`,
			Module:    RootModuleInstance.Child("bloop", NoKey),
			WantFrom:  RootModuleInstance.Child("bloop", NoKey).Child("foo", IntKey(0)),
			WantTo:    RootModuleInstance.Child("bloop", NoKey).Child("foo", NoKey),
		},
		{
			InputFrom: `module.foo`,
			InputTo:   `foo.bar`,
			Module:    RootModuleInstance,
			WantFrom:  nil, // Can't unify module call with resource
			WantTo:    nil,
		},
		{
			InputFrom: `module.foo[0]`,
			InputTo:   `foo.bar`,
			Module:    RootModuleInstance,
			WantFrom:  nil, // Can't unify module instance with resource
			WantTo:    nil,
		},
		{
			InputFrom: `module.foo`,
			InputTo:   `foo.bar[0]`,
			Module:    RootModuleInstance,
			WantFrom:  nil, // Can't unify module call with resource instance
			WantTo:    nil,
		},
		{
			InputFrom: `module.foo[0]`,
			InputTo:   `foo.bar[0]`,
			Module:    RootModuleInstance,
			WantFrom:  nil, // Can't unify module instance with resource instance
			WantTo:    nil,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s to %s in %s", test.InputFrom, test.InputTo, test.Module), func(t *testing.T) {
			parseInput := func(input string) *MoveEndpoint {
				t.Helper()

				traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(input), "", hcl.InitialPos)
				if hclDiags.HasErrors() {
					// We're not trying to test the HCL parser here, so any
					// failures at this point are likely to be bugs in the
					// test case itself.
					t.Fatalf("syntax error: %s", hclDiags.Error())
				}

				moveEp, diags := ParseMoveEndpoint(traversal)
				if diags.HasErrors() {
					t.Fatalf("unexpected error: %s", diags.Err().Error())
				}
				return moveEp
			}

			fromEp := parseInput(test.InputFrom)
			toEp := parseInput(test.InputTo)

			diffOpts := cmpopts.IgnoreUnexported(ModuleCall{})
			gotFrom, gotTo := UnifyMoveEndpoints(test.Module, fromEp, toEp)
			if diff := cmp.Diff(test.WantFrom, gotFrom, diffOpts); diff != "" {
				t.Errorf("wrong 'from' address\n%s", diff)
			}
			if diff := cmp.Diff(test.WantTo, gotTo, diffOpts); diff != "" {
				t.Errorf("wrong 'to' address\n%s", diff)
			}
		})
	}
}

func TestMoveEndpointConfigMoveable(t *testing.T) {
	tests := []struct {
		Input  string
		Module Module
		Want   ConfigMoveable
	}{
		{
			`foo.bar`,
			RootModule,
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "bar",
				},
			},
		},
		{
			`foo.bar[0]`,
			RootModule,
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "bar",
				},
			},
		},
		{
			`module.foo.bar.baz`,
			RootModule,
			ConfigResource{
				Module: Module{"foo"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "bar",
					Name: "baz",
				},
			},
		},
		{
			`module.foo[0].bar.baz`,
			RootModule,
			ConfigResource{
				Module: Module{"foo"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "bar",
					Name: "baz",
				},
			},
		},
		{
			`foo.bar`,
			Module{"boop"},
			ConfigResource{
				Module: Module{"boop"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "bar",
				},
			},
		},
		{
			`module.bloop.foo.bar`,
			Module{"bleep"},
			ConfigResource{
				Module: Module{"bleep", "bloop"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "foo",
					Name: "bar",
				},
			},
		},
		{
			`module.foo.bar.baz`,
			RootModule,
			ConfigResource{
				Module: Module{"foo"},
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "bar",
					Name: "baz",
				},
			},
		},
		{
			`module.foo`,
			RootModule,
			Module{"foo"},
		},
		{
			`module.foo[0]`,
			RootModule,
			Module{"foo"},
		},
		{
			`module.bloop`,
			Module{"bleep"},
			Module{"bleep", "bloop"},
		},
		{
			`module.bloop[0]`,
			Module{"bleep"},
			Module{"bleep", "bloop"},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s in %s", test.Input, test.Module), func(t *testing.T) {
			traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(test.Input), "", hcl.InitialPos)
			if hclDiags.HasErrors() {
				// We're not trying to test the HCL parser here, so any
				// failures at this point are likely to be bugs in the
				// test case itself.
				t.Fatalf("syntax error: %s", hclDiags.Error())
			}

			moveEp, diags := ParseMoveEndpoint(traversal)
			if diags.HasErrors() {
				t.Fatalf("unexpected error: %s", diags.Err().Error())
			}

			got := moveEp.ConfigMoveable(test.Module)
			if diff := cmp.Diff(test.Want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
