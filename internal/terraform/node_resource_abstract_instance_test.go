// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestNodeAbstractResourceInstanceProvider(t *testing.T) {
	tests := []struct {
		Addr                 addrs.AbsResourceInstance
		Config               *configs.Resource
		StoredProviderConfig addrs.AbsProviderConfig
		Want                 addrs.Provider
	}{
		{
			Addr: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			Want: addrs.Provider{
				Hostname:  addrs.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "null",
			},
		},
		{
			Addr: addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "terraform_remote_state",
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			Want: addrs.Provider{
				// As a special case, the type prefix "terraform_" maps to
				// the builtin provider, not the default one.
				Hostname:  addrs.BuiltInProviderHost,
				Namespace: addrs.BuiltInProviderNamespace,
				Type:      "terraform",
			},
		},
		{
			Addr: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			Config: &configs.Resource{
				// Just enough configs.Resource for the Provider method. Not
				// actually valid for general use.
				Provider: addrs.Provider{
					Hostname:  addrs.DefaultProviderRegistryHost,
					Namespace: "awesomecorp",
					Type:      "happycloud",
				},
			},
			// The config overrides the default behavior.
			Want: addrs.Provider{
				Hostname:  addrs.DefaultProviderRegistryHost,
				Namespace: "awesomecorp",
				Type:      "happycloud",
			},
		},
		{
			Addr: addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "terraform_remote_state",
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			Config: &configs.Resource{
				// Just enough configs.Resource for the Provider method. Not
				// actually valid for general use.
				Provider: addrs.Provider{
					Hostname:  addrs.DefaultProviderRegistryHost,
					Namespace: "awesomecorp",
					Type:      "happycloud",
				},
			},
			// The config overrides the default behavior.
			Want: addrs.Provider{
				Hostname:  addrs.DefaultProviderRegistryHost,
				Namespace: "awesomecorp",
				Type:      "happycloud",
			},
		},
		{
			Addr: addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "null_resource",
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			Config: nil,
			StoredProviderConfig: addrs.AbsProviderConfig{
				Module: addrs.RootModule,
				Provider: addrs.Provider{
					Hostname:  addrs.DefaultProviderRegistryHost,
					Namespace: "awesomecorp",
					Type:      "null",
				},
			},
			// The stored provider config overrides the default behavior.
			Want: addrs.Provider{
				Hostname:  addrs.DefaultProviderRegistryHost,
				Namespace: "awesomecorp",
				Type:      "null",
			},
		},
	}

	for _, test := range tests {
		var name string
		if test.Config != nil {
			name = fmt.Sprintf("%s with configured %s", test.Addr, test.Config.Provider)
		} else {
			name = fmt.Sprintf("%s with no configuration", test.Addr)
		}
		t.Run(name, func(t *testing.T) {
			node := &NodeAbstractResourceInstance{
				// Just enough NodeAbstractResourceInstance for the Provider
				// function. (This would not be valid for some other functions.)
				Addr: test.Addr,
				NodeAbstractResource: NodeAbstractResource{
					Addr:                 test.Addr.ConfigResource(),
					Config:               test.Config,
					storedProviderConfig: test.StoredProviderConfig,
				},
			}
			got := node.Provider()
			if got != test.Want {
				t.Errorf("wrong result\naddr:  %s\nconfig: %#v\ngot:   %s\nwant:  %s", test.Addr, test.Config, got, test.Want)
			}
		})
	}
}

func TestNodeAbstractResourceInstance_WriteResourceInstanceState(t *testing.T) {
	state := states.NewState()
	ctx := new(MockEvalContext)
	ctx.StateState = state.SyncWrapper()
	ctx.Scope = evalContextModuleInstance{Addr: addrs.RootModuleInstance}

	mockProvider := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Optional: true,
			},
		},
	})

	obj := &states.ResourceInstanceObject{
		Value: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("i-abc123"),
		}),
		Status: states.ObjectReady,
	}

	node := &NodeAbstractResourceInstance{
		Addr: mustResourceInstanceAddr("aws_instance.foo"),
		// instanceState:        obj,
		NodeAbstractResource: NodeAbstractResource{
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}
	ctx.ProviderProvider = mockProvider
	ctx.ProviderSchemaSchema = mockProvider.GetProviderSchema()

	err := node.writeResourceInstanceState(ctx, obj, workingState)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	checkStateString(t, state, `
aws_instance.foo:
  ID = i-abc123
  provider = provider["registry.terraform.io/hashicorp/aws"]
	`)
}

func TestNodeAbstractResourceInstance_refresh_with_deferred_read(t *testing.T) {
	state := states.NewState()
	evalCtx := &MockEvalContext{}
	evalCtx.StateState = state.SyncWrapper()
	evalCtx.Scope = evalContextModuleInstance{Addr: addrs.RootModuleInstance}

	mockProvider := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Optional: true,
			},
		},
	})
	mockProvider.ConfigureProviderCalled = true

	mockProvider.ReadResourceFn = func(providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{
			NewState: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}),
			Deferred: &providers.Deferred{
				Reason: providers.DeferredReasonAbsentPrereq,
			},
		}
	}

	obj := &states.ResourceInstanceObject{
		Value: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("i-abc123"),
		}),
		Status: states.ObjectReady,
	}

	node := &NodeAbstractResourceInstance{
		Addr: mustResourceInstanceAddr("aws_instance.foo"),
		NodeAbstractResource: NodeAbstractResource{
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}
	evalCtx.ProviderProvider = mockProvider
	evalCtx.ProviderSchemaSchema = mockProvider.GetProviderSchema()
	evalCtx.DeferralsState = deferring.NewDeferred(true)

	rio, deferred, diags := node.refresh(evalCtx, states.NotDeposed, obj, true)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	value := rio.Value
	if value.IsWhollyKnown() {
		t.Fatalf("value was known: %v", value)
	}

	if deferred == nil {
		t.Fatalf("expected deferral to be present")
	}

	if deferred.Reason != providers.DeferredReasonAbsentPrereq {
		t.Fatalf("expected deferral to be AbsentPrereq, got %s", deferred.Reason)
	}
}

func TestNodeAbstractResourceInstance_apply_with_unknown_values(t *testing.T) {
	state := states.NewState()
	evalCtx := &MockEvalContext{}
	evalCtx.StateState = state.SyncWrapper()
	evalCtx.Scope = evalContextModuleInstance{Addr: addrs.RootModuleInstance}

	mockProvider := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Optional: true,
			},
		},
	})
	mockProvider.ConfigureProviderCalled = true

	node := &NodeAbstractResourceInstance{
		Addr: mustResourceInstanceAddr("aws_instance.foo"),
		NodeAbstractResource: NodeAbstractResource{
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}
	evalCtx.ProviderProvider = mockProvider
	evalCtx.ProviderSchemaSchema = mockProvider.GetProviderSchema()
	evalCtx.EvaluateBlockResult = cty.ObjectVal(map[string]cty.Value{
		"id": cty.UnknownVal(cty.String),
	})
	priorState := &states.ResourceInstanceObject{
		Value: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("prior"),
		}),
		Status: states.ObjectReady,
	}
	change := &plans.ResourceInstanceChange{
		Addr: node.Addr,
		Change: plans.Change{
			Action: plans.Update,
			Before: priorState.Value,
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}),
		},
	}

	// Not needed for this test
	applyConfig := &configs.Resource{}
	keyData := instances.RepetitionData{}

	newState, diags := node.apply(evalCtx, priorState, change, applyConfig, keyData, false)

	tfdiags.AssertDiagnosticsMatch(t, diags, tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Configuration contains unknown value",
		Detail:   "configuration for aws_instance.foo still contains unknown values during apply (this is a bug in Terraform; please report it!)\nThe following paths in the resource configuration are unknown:\n.id",
	}))

	if !newState.Value.RawEquals(priorState.Value) {
		t.Fatalf("expected prior state to be preserved, got %s", newState.Value.GoString())
	}
}
