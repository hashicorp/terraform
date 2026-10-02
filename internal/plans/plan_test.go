// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/zclconf/go-cty/cty"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
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

func TestBackend_Validate(t *testing.T) {

	typeName := "foobar"
	workspace := "default"
	config := cty.ObjectVal(map[string]cty.Value{
		"bool": cty.BoolVal(true),
	})
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"bool": {
				Type: cty.Bool,
			},
		},
	}

	// Not-empty cases
	t.Run("backend is not valid if all values are set", func(t *testing.T) {
		b, err := NewBackend(typeName, config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := b.Validate(); err != nil {
			t.Fatalf("expected the Backend to be valid, given all input values were provided: %s", err)
		}
	})
	t.Run("backend is not empty if the schema contains no attributes or blocks", func(t *testing.T) {
		emptyConfig := cty.ObjectVal(map[string]cty.Value{})
		emptySchema := &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				// No attributes
			},
		}
		b, err := NewBackend(typeName, emptyConfig, emptySchema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := b.Validate(); err != nil {
			t.Fatalf("expected the Backend to be valid, as empty schemas should be tolerated: %s", err)
		}
	})

	// Empty cases
	t.Run("backend is empty if type name is missing", func(t *testing.T) {
		b, err := NewBackend("", config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := b.Validate(); err == nil {
			t.Fatalf("expected the Backend to be invalid, given the type being unset: %#v", b)
		}
	})
	t.Run("backend is empty if workspace name is missing", func(t *testing.T) {
		b, err := NewBackend(typeName, config, schema, "")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := b.Validate(); err == nil {
			t.Fatalf("expected the Backend to be invalid, given the type being unset: %#v", b)
		}
	})
}

func TestStateStore_Validate(t *testing.T) {
	typeName := "test_store"
	providerVersion := version.Must(version.NewSemver("1.2.3"))
	source := addrs.MustParseProviderSourceString("hashicorp/test")
	workspace := "default"
	config := cty.ObjectVal(map[string]cty.Value{
		"bool": cty.BoolVal(true),
	})
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"bool": {
				Type: cty.Bool,
			},
		},
	}

	// Not-empty cases
	t.Run("state store is not empty if all values are set", func(t *testing.T) {
		s, err := NewStateStore(typeName, providerVersion, &source, config, schema, config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected the StateStore to be valid, given all input values were provided: %s", err)
		}
	})
	t.Run("state store is not empty if the state store config is present but contains all null values", func(t *testing.T) {
		nullConfig := cty.ObjectVal(map[string]cty.Value{
			"bool": cty.NullVal(cty.Bool),
		})
		s, err := NewStateStore(typeName, providerVersion, &source, nullConfig, schema, config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected the StateStore to be valid, despite the state store config containing only null values: %s", err)
		}
	})
	t.Run("state store is not empty if the provider config is present but contains all null values", func(t *testing.T) {
		nullConfig := cty.ObjectVal(map[string]cty.Value{
			"bool": cty.NullVal(cty.Bool),
		})
		s, err := NewStateStore(typeName, providerVersion, &source, nullConfig, schema, config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected the StateStore to be valid, despite the provider config containing only null values:  %s", err)
		}
	})
	t.Run("state store is not incorrectly identified as empty if the state store's schema contains no attributes or blocks", func(t *testing.T) {
		emptyConfig := cty.ObjectVal(map[string]cty.Value{})
		emptySchema := &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				// No attributes
			},
		}
		s, err := NewStateStore(typeName, providerVersion, &source, emptyConfig, emptySchema, config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected the StateStore to be valid, as empty schemas should be tolerated: %s", err)
		}
	})
	t.Run("state store is not incorrectly identified as empty if the provider's schema contains no attributes or blocks", func(t *testing.T) {
		emptyConfig := cty.ObjectVal(map[string]cty.Value{})
		emptySchema := &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				// No attributes
			},
		}
		s, err := NewStateStore(typeName, providerVersion, &source, config, schema, emptyConfig, emptySchema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected the StateStore to be valid, as empty schemas should be tolerated: %s", err)
		}
	})

	// Empty cases
	t.Run("state store is empty if the type is missing", func(t *testing.T) {
		s, err := NewStateStore("", providerVersion, &source, config, schema, config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err == nil {
			t.Fatalf("expected the StateStore to be invalid,  given the type name is missing: %s", err)
		}
	})
	t.Run("state store is empty if the provider version is missing", func(t *testing.T) {
		s, err := NewStateStore(typeName, nil, &source, config, schema, config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err == nil {
			t.Fatalf("expected the StateStore to be invalid, given the version is missing: %s", err)
		}
	})
	t.Run("state store is empty if the provider source is missing", func(t *testing.T) {
		s, err := NewStateStore(typeName, providerVersion, nil, config, schema, config, schema, workspace)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err == nil {
			t.Fatalf("expected the StateStore to be invalid, given the version is missing: %s", err)
		}
	})
	t.Run("state store is empty if the workspace name is missing", func(t *testing.T) {
		s, err := NewStateStore(typeName, providerVersion, &source, cty.NilVal, schema, config, schema, "")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if err := s.Validate(); err == nil {
			t.Fatalf("expected the StateStore to be invalid, given the workspace name is missing: %s", err)
		}
	})
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
