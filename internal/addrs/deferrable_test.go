package addrs

import (
	"fmt"
	"testing"
)

func TestDeferrableString(t *testing.T) {
	tests := []struct {
		Addr Deferrable
		Want string
	}{
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			}.Instance(NoKey).Absolute(RootModuleInstance),
			`foo.bar`,
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			}.Instance(IntKey(2)).Absolute(RootModuleInstance),
			`foo.bar[2]`,
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			}.Instance(StringKey("blub")).Absolute(RootModuleInstance),
			`foo.bar["blub"]`,
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			}.Instance(NoKey).Absolute(RootModuleInstance.Child("boop", NoKey)),
			`module.boop.foo.bar`,
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			}.Instance(NoKey).Absolute(RootModuleInstance.Child("boop", IntKey(6))),
			`module.boop[6].foo.bar`,
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			}.Instance(NoKey).Absolute(RootModuleInstance.Child("boop", StringKey("a"))),
			`module.boop["a"].foo.bar`,
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			}.InModule(RootModule),
			`foo.bar[*]`,
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			}.InModule(RootModule.Child("boop")),
			`module.boop[*].foo.bar[*]`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Addr), func(t *testing.T) {
			got := test.Addr.DeferrableString()

			if got != test.Want {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, test.Want)
			}
		})
	}
}
