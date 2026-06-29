// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// listPolicyTestResourceSchema returns a minimal configschema.Block for the
// test resource type. Attributes are Optional-only to avoid ExtractLegacyConfigFromState
// filtering them out on the fallback path.
func listPolicyTestResourceSchema() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"instance_type": {Type: cty.String, Optional: true},
			"ami":           {Type: cty.String, Optional: true},
		},
	}
}

// listPolicyTestProviderSchema returns a providers.ProviderSchema for
// "test_resource" with ServerCapabilities.GenerateResourceConfig set as given.
func listPolicyTestProviderSchema(generateResourceConfig bool) providers.ProviderSchema {
	return providers.ProviderSchema{
		ResourceTypes: map[string]providers.Schema{
			"test_resource": {Body: listPolicyTestResourceSchema()},
		},
		ServerCapabilities: providers.ServerCapabilities{
			GenerateResourceConfig: generateResourceConfig,
		},
	}
}

// listPolicyTestNode constructs a minimal NodePlannableResourceInstance for a
// NoKey list block at the root module.
func listPolicyTestNode(resourceType, name string) *NodePlannableResourceInstance {
	resolvedProvider := addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("test"),
		Module:   addrs.RootModule,
	}
	listAddr := addrs.Resource{
		Mode: addrs.ListResourceMode,
		Type: resourceType,
		Name: name,
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	return &NodePlannableResourceInstance{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			NodeAbstractResource: NodeAbstractResource{
				ResolvedProvider: resolvedProvider,
				Config: &configs.Resource{
					Mode:   addrs.ListResourceMode,
					Type:   resourceType,
					Name:   name,
					Config: hcl.EmptyBody(),
				},
			},
			Addr: listAddr,
		},
	}
}

// listPolicyTestContext constructs a MockEvalContext with the given provider
// and schema. The list block address is registered in the expander as a
// singleton so ResourceExpansionEnum does not panic.
func listPolicyTestContext(listBlockAddr addrs.AbsResourceInstance, p providers.Interface, schema providers.ProviderSchema) *MockEvalContext {
	expander := instances.NewExpander(nil)
	expander.SetResourceSingle(
		listBlockAddr.Module,
		listBlockAddr.Resource.Resource,
	)
	return &MockEvalContext{
		InstanceExpanderExpander: expander,
		ProviderProvider:         p,
		ProviderSchemaSchema:     schema,
	}
}

// listPolicyTestElement returns a list response element with both "state" and
// "identity" attributes present (include_resource = true).
func listPolicyTestElement(stateVal, identityVal cty.Value) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"state":    stateVal,
		"identity": identityVal,
	})
}

// listPolicyTestElementNoState returns a list response element with only an
// "identity" attribute — no "state" — representing include_resource = false.
func listPolicyTestElementNoState(identityVal cty.Value) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"identity": identityVal,
	})
}

// TestGenerateListResourcePolicyData_ProviderRPCPath verifies that when the
// provider advertises ServerCapabilities.GenerateResourceConfig, the config is
// derived from the GenerateResourceConfig RPC rather than state extraction.
func TestGenerateListResourcePolicyData_ProviderRPCPath(t *testing.T) {
	// The RPC returns a config that differs from what ExtractLegacyConfigFromState
	// would produce, proving the RPC path was taken.
	rpcConfig := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("m5.large"),
		"ami":           cty.StringVal("ami-rpc"),
	})

	rpcCalled := false
	p := &testing_provider.MockProvider{}
	p.GenerateResourceConfigFn = func(_ providers.GenerateResourceConfigRequest) providers.GenerateResourceConfigResponse {
		rpcCalled = true
		return providers.GenerateResourceConfigResponse{Config: rpcConfig}
	}

	schema := listPolicyTestProviderSchema(true)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr
	ctx := listPolicyTestContext(listBlockAddr, p, schema)

	stateVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-123")})
	data := cty.TupleVal([]cty.Value{listPolicyTestElement(stateVal, identityVal)})

	results, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)

	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]

	if got.Unknown {
		t.Error("expected Unknown = false for resource with state")
	}
	if !rpcCalled {
		t.Error("expected GenerateResourceConfig RPC to be called")
	}
	if !got.GeneratedConfig.RawEquals(rpcConfig) {
		t.Errorf("GeneratedConfig: got %#v, want %#v", got.GeneratedConfig, rpcConfig)
	}
	if got.ResourceConfig != n.Config {
		t.Error("ResourceConfig should point to n.Config")
	}
	if got.ListBlockAddr.String() != listBlockAddr.String() {
		t.Errorf("ListBlockAddr: got %s, want %s", got.ListBlockAddr, listBlockAddr)
	}
}

// TestGenerateListResourcePolicyData_LegacyFallbackPath verifies that when the
// provider does not advertise GenerateResourceConfig, config is derived via
// genconfig.ExtractLegacyConfigFromState without calling the RPC.
func TestGenerateListResourcePolicyData_LegacyFallbackPath(t *testing.T) {
	p := &testing_provider.MockProvider{}
	// GenerateResourceConfigResponse is intentionally not set. Calling the RPC
	// would panic, so a successful test also proves the RPC was not invoked.

	schema := listPolicyTestProviderSchema(false)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr
	ctx := listPolicyTestContext(listBlockAddr, p, schema)

	stateVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-123")})
	data := cty.TupleVal([]cty.Value{listPolicyTestElement(stateVal, identityVal)})

	results, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)

	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]

	if got.Unknown {
		t.Error("expected Unknown = false for resource with state")
	}
	if got.GeneratedConfig == cty.NilVal {
		t.Error("expected non-nil GeneratedConfig from the fallback path")
	}
}

// TestGenerateListResourcePolicyData_MultipleResources verifies that a list
// block returning N elements produces exactly N policy inputs, all with
// correct metadata.
func TestGenerateListResourcePolicyData_MultipleResources(t *testing.T) {
	const count = 3

	p := &testing_provider.MockProvider{}
	schema := listPolicyTestProviderSchema(false)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr
	ctx := listPolicyTestContext(listBlockAddr, p, schema)

	elements := make([]cty.Value, count)
	for i := range elements {
		elements[i] = listPolicyTestElement(
			cty.ObjectVal(map[string]cty.Value{
				"instance_type": cty.StringVal(fmt.Sprintf("t2.micro-%d", i)),
				"ami":           cty.StringVal(fmt.Sprintf("ami-%d", i)),
			}),
			cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal(fmt.Sprintf("i-%d", i))}),
		)
	}
	data := cty.TupleVal(elements)

	results, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)

	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	if len(results) != count {
		t.Fatalf("expected %d results, got %d", count, len(results))
	}
	for i, r := range results {
		if r.Unknown {
			t.Errorf("result %d: expected Unknown = false", i)
		}
		if r.ListBlockAddr.String() != listBlockAddr.String() {
			t.Errorf("result %d: ListBlockAddr = %s, want %s", i, r.ListBlockAddr, listBlockAddr)
		}
		if r.ResourceConfig != n.Config {
			t.Errorf("result %d: ResourceConfig should point to n.Config", i)
		}
	}
}

// TestGenerateListResourcePolicyData_IncludeResourceFalse verifies that a
// list response element with no "state" attribute is recorded as Unknown with
// a "Policy evaluation skipped" warning, and config generation is not attempted.
func TestGenerateListResourcePolicyData_IncludeResourceFalse(t *testing.T) {
	p := &testing_provider.MockProvider{}
	schema := listPolicyTestProviderSchema(false)
	n := listPolicyTestNode("test_resource", "mylist")
	listBlockAddr := n.Addr
	ctx := listPolicyTestContext(listBlockAddr, p, schema)

	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-123")})
	data := cty.TupleVal([]cty.Value{listPolicyTestElementNoState(identityVal)})

	results, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)

	// Soft-error: no function-level errors; the Unknown is recorded in the result.
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]

	if !got.Unknown {
		t.Error("expected Unknown = true for resource without state")
	}
	if got.UnknownReason != unknownReasonNoState {
		t.Errorf("UnknownReason = %v, want unknownReasonNoState", got.UnknownReason)
	}
	if got.GeneratedConfig != cty.NilVal {
		t.Errorf("expected zero GeneratedConfig when Unknown = true, got %#v", got.GeneratedConfig)
	}

	var hasSkipWarning bool
	for _, d := range diags {
		if d.Severity() == tfdiags.Warning && d.Description().Summary == "Policy evaluation skipped" {
			hasSkipWarning = true
			break
		}
	}
	if !hasSkipWarning {
		t.Error(`expected "Policy evaluation skipped" warning in returned diags`)
	}
}

// TestGenerateListResourcePolicyData_SyntheticAddressFormat verifies that the
// synthetic managed-mode address assigned to each discovered resource matches
// the formula used by genconfig.GenerateListResourceContents:
//
//	no key: <name>_<idx>
//	keyed:  <name>_<expansionEnum>_<idx>
func TestGenerateListResourcePolicyData_SyntheticAddressFormat(t *testing.T) {
	stateVal := cty.ObjectVal(map[string]cty.Value{
		"instance_type": cty.StringVal("t2.micro"),
		"ami":           cty.StringVal("ami-12345"),
	})
	identityVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-1")})

	t.Run("no_key", func(t *testing.T) {
		p := &testing_provider.MockProvider{}
		schema := listPolicyTestProviderSchema(false)
		n := listPolicyTestNode("test_resource", "mylist")
		listBlockAddr := n.Addr
		ctx := listPolicyTestContext(listBlockAddr, p, schema)

		data := cty.TupleVal([]cty.Value{
			listPolicyTestElement(stateVal, identityVal),
			listPolicyTestElement(stateVal, identityVal),
		})

		results, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}

		for i, wantName := range []string{"mylist_0", "mylist_1"} {
			addr := results[i].SyntheticAddr.Resource.Resource
			if addr.Name != wantName {
				t.Errorf("result %d: synthetic name = %q, want %q", i, addr.Name, wantName)
			}
			if addr.Mode != addrs.ManagedResourceMode {
				t.Errorf("result %d: expected ManagedResourceMode, got %s", i, addr.Mode)
			}
			if addr.Type != "test_resource" {
				t.Errorf("result %d: type = %q, want %q", i, addr.Type, "test_resource")
			}
		}
	})

	t.Run("keyed", func(t *testing.T) {
		// Two for_each keys: "a" (enum 0) and "b" (enum 1, sorted order).
		// Testing the "b" instance: with one result element at idx 0 the
		// synthetic name must be "mylist_1_0".
		resolvedProvider := addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		}
		resourceAddr := addrs.Resource{
			Mode: addrs.ListResourceMode,
			Type: "test_resource",
			Name: "mylist",
		}
		listBlockAddr := resourceAddr.Instance(addrs.StringKey("b")).Absolute(addrs.RootModuleInstance)

		n := &NodePlannableResourceInstance{
			NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
				NodeAbstractResource: NodeAbstractResource{
					ResolvedProvider: resolvedProvider,
					Config: &configs.Resource{
						Mode:   addrs.ListResourceMode,
						Type:   "test_resource",
						Name:   "mylist",
						Config: hcl.EmptyBody(),
					},
				},
				Addr: listBlockAddr,
			},
		}

		p := &testing_provider.MockProvider{}
		schema := listPolicyTestProviderSchema(false)

		expander := instances.NewExpander(nil)
		expander.SetResourceForEach(
			addrs.RootModuleInstance,
			resourceAddr,
			map[string]cty.Value{
				"a": cty.StringVal("a"),
				"b": cty.StringVal("b"),
			},
		)
		ctx := &MockEvalContext{
			InstanceExpanderExpander: expander,
			ProviderProvider:         p,
			ProviderSchemaSchema:     schema,
		}

		data := cty.TupleVal([]cty.Value{listPolicyTestElement(stateVal, identityVal)})

		results, diags := n.generateListResourcePolicyData(ctx, listBlockAddr, data)
		if diags.HasErrors() {
			t.Fatalf("unexpected errors: %s", diags.Err())
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}

		wantName := "mylist_1_0"
		gotName := results[0].SyntheticAddr.Resource.Resource.Name
		if gotName != wantName {
			t.Errorf("keyed synthetic name = %q, want %q", gotName, wantName)
		}
	})
}
