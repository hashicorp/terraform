// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestProviderAddrs(t *testing.T) {
	// Inputs for plan
	provider := &Provider{}
	err := provider.SetSource("registry.terraform.io/hashicorp/pluggable")
	if err != nil {
		panic(err)
	}
	err = provider.SetVersion("9.9.9")
	if err != nil {
		panic(err)
	}
	config, err := NewDynamicValue(cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	}), cty.Object(map[string]cty.Type{
		"foo": cty.String,
	}))
	if err != nil {
		panic(err)
	}
	provider.Config = config

	// Prepare plan
	plan := &Plan{
		StateStore: &StateStore{
			Type:      "pluggable_foobar",
			Provider:  provider,
			Config:    config,
			Workspace: "default",
		},
		VariableValues: map[string]DynamicValue{},
		Changes: &ChangesSrc{
			Resources: []*ResourceInstanceChangeSrc{
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					},
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					DeposedKey: "foodface",
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					},
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "what",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule.Child("foo"),
						Provider: addrs.NewDefaultProvider("test"),
					},
				},
			},
		},
	}

	got := plan.ProviderAddrs()
	want := []addrs.AbsProviderConfig{
		// Providers used for managed resources
		{
			Module:   addrs.RootModule.Child("foo"),
			Provider: addrs.NewDefaultProvider("test"),
		},
		{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("test"),
		},
		// Provider used for pluggable state storage
		{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("pluggable"),
		},
	}

	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

// Module outputs should not effect the result of Empty
func TestModuleOutputChangesEmpty(t *testing.T) {
	changes := &ChangesSrc{
		Outputs: []*OutputChangeSrc{
			{
				Addr: addrs.AbsOutputValue{
					Module: addrs.RootModuleInstance.Child("child", addrs.NoKey),
					OutputValue: addrs.OutputValue{
						Name: "output",
					},
				},
				ChangeSrc: ChangeSrc{
					Action: Update,
					Before: []byte("a"),
					After:  []byte("b"),
				},
			},
		},
	}

	if !changes.Empty() {
		t.Fatal("plan has no visible changes")
	}
}
