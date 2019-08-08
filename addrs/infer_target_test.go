package addrs

import (
	"testing"

	"github.com/go-test/deep"
)

func TestInferAbsResourceInstanceStr(t *testing.T) {
	tests := []struct {
		Input   string
		Want    AbsResourceInstance
		WantErr string
	}{
		{
			`test_resource.foo`,
			AbsResourceInstance{
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "foo",
					},
					Key: NoKey,
				},
			},
			``,
		},
		{
			`test_resource.foo[1]`,
			AbsResourceInstance{
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "foo",
					},
					Key: IntKey(1),
				},
			},
			``,
		},
		{
			`test_resource.foo[bar]`,
			AbsResourceInstance{
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "foo",
					},
					Key: StringKey("bar"),
				},
			},
			``,
		},
		{
			`test_resource.foo["bar"]`,
			AbsResourceInstance{
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "foo",
					},
					Key: StringKey("bar"),
				},
			},
			``,
		},
		{
			`module.foo.test_resource.bar`,
			AbsResourceInstance{
				Module: ModuleInstance{
					{
						Name: "foo",
					},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "bar",
					},
				},
			},
			``,
		},
		{
			`module.foo.test_resource.bar[1]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					{
						Name: "foo",
					},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "bar",
					},
					Key: IntKey(1),
				},
			},
			``,
		},
		{
			`module.foo.test_resource.bar[baz]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					{
						Name: "foo",
					},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "bar",
					},
					Key: StringKey("baz"),
				},
			},
			``,
		},
		{
			`module.foo.test_resource.bar["baz"]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					{
						Name: "foo",
					},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "bar",
					},
					Key: StringKey("baz"),
				},
			},
			``,
		},
		{
			`module.foo[1].test_resource.bar`,
			AbsResourceInstance{
				Module: ModuleInstance{
					{
						Name:        "foo",
						InstanceKey: IntKey(1),
					},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "bar",
					},
				},
			},
			``,
		},
		{
			`module.foo[1].test_resource.bar[2]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					{
						Name:        "foo",
						InstanceKey: IntKey(1),
					},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "bar",
					},
					Key: IntKey(2),
				},
			},
			``,
		},
		{
			`module.foo[1].test_resource.bar[baz]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					{
						Name:        "foo",
						InstanceKey: IntKey(1),
					},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "bar",
					},
					Key: StringKey("baz"),
				},
			},
			``,
		},
		{
			`module.foo[1].test_resource.bar["baz"]`,
			AbsResourceInstance{
				Module: ModuleInstance{
					{
						Name:        "foo",
						InstanceKey: IntKey(1),
					},
				},
				Resource: ResourceInstance{
					Resource: Resource{
						Mode: ManagedResourceMode,
						Type: "test_resource",
						Name: "bar",
					},
					Key: StringKey("baz"),
				},
			},
			``,
		},
		{
			`aws_instance`,
			AbsResourceInstance{},
			`Resource specification must include a resource type and name.`,
		},
		{
			`module`,
			AbsResourceInstance{},
			`Prefix "module." must be followed by a module name.`,
		},
		{
			`module["baz"]`,
			AbsResourceInstance{},
			`Prefix "module." must be followed by a module name.`,
		},
		{
			`module.baz.bar`,
			AbsResourceInstance{},
			`Resource specification must include a resource type and name.`,
		},
		{
			`aws_instance.foo.bar`,
			AbsResourceInstance{},
			`Resource instance key must be given in square brackets.`,
		},
		{
			`aws_instance.foo[1].baz`,
			AbsResourceInstance{},
			`Unexpected extra operators after address.`,
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			addr, addrDiags := InferAbsResourceInstanceStr(test.Input)

			switch len(addrDiags) {
			case 0:
				if test.WantErr != "" {
					t.Fatalf("succeeded; want error: %s", test.WantErr)
				}
			case 1:
				if test.WantErr == "" {
					t.Fatalf("unexpected diagnostics: %s", addrDiags.Err())
				}
				if got, want := addrDiags[0].Description().Detail, test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
			default:
				t.Fatalf("too many diagnostics: %s", addrDiags.Err())
			}

			if addrDiags.HasErrors() {
				return
			}

			for _, problem := range deep.Equal(addr, test.Want) {
				t.Errorf(problem)
			}
		})
	}
}
