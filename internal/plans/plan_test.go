// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// TestProviderAddrs_basic tests sourcing providers from a plan file that only uses providers
// for resource management only.
func TestProviderAddrs_basic(t *testing.T) {

	// Prepare plan
	plan := &Plan{
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
	}

	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

// TestProviderAddrs_withStateStore tests sourcing providers from a plan file that uses providers
// both for resource management and state storage.
func TestProviderAddrs_withStateStore(t *testing.T) {
	// Inputs for plan
	provider := &Provider{}
	err := provider.SetSource("registry.terraform.io/hashicorp/pluggable")
	if err != nil {
		t.Fatal(err)
	}
	err = provider.SetVersion("9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	config, err := NewDynamicValue(cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	}), cty.Object(map[string]cty.Type{
		"foo": cty.String,
	}))
	if err != nil {
		t.Fatal(err)
	}

	// Prepare plan
	plan := &Plan{
		StateStore: StateStore{
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
