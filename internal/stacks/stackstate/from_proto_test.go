// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestLoader_basic(t *testing.T) {
	aComponentInstAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{
				Name: "a",
			},
		},
	}
	aResourceInstAddr := stackaddrs.AbsResourceInstance{
		Component: aComponentInstAddr,
		Item: addrs.AbsResourceInstance{
			Module: addrs.RootModuleInstance,
			Resource: addrs.ResourceInstance{
				Resource: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test",
					Name: "foo",
				},
			},
		},
	}
	providerAddr := addrs.NewBuiltInProvider("test")
	providerInstAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: providerAddr,
	}

	loader := NewLoader()
	loader.AddDirectProto(
		statekeys.String(statekeys.ComponentInstance{
			ComponentInstanceAddr: aComponentInstAddr,
		}),
		&tfstackdata1.StateComponentInstanceV1{
			OutputValues: make(map[string]*tfstackdata1.DynamicValue),
		},
	)
	attrs := `
    {
        "for_module": "a",
        "arg": null,
        "result": "result for \"a\""
	}
`
	loader.AddDirectProto(
		statekeys.String(statekeys.ResourceInstanceObject{
			ResourceInstance: aResourceInstAddr,
		}),
		&tfstackdata1.StateResourceInstanceObjectV1{
			Status:             tfstackdata1.StateResourceInstanceObjectV1_READY,
			ProviderConfigAddr: providerInstAddr.String(),
			ValueJson:          []byte(attrs),
		},
	)
	state := loader.State()

	if !state.HasComponentInstance(aComponentInstAddr) {
		t.Errorf("component instance %s not found in state", aComponentInstAddr)
	}

	got := state.ResourceInstanceObjectSrc(
		stackaddrs.AbsResourceInstanceObject{
			Component: aComponentInstAddr,
			Item:      aResourceInstAddr.Item.CurrentObject(),
		},
	)
	want := &states.ResourceInstanceObjectSrc{
		AttrsJSON:          []byte(attrs),
		AttrSensitivePaths: []cty.Path{},
		Status:             states.ObjectReady,
	}

	if diff := cmp.Diff(got, want, cmpopts.IgnoreUnexported(states.ResourceInstanceObjectSrc{})); diff != "" {
		t.Errorf("unexpected resource instance object\ndiff: %s", diff)
	}
}

func TestLoader_consumed(t *testing.T) {
	loader := NewLoader()
	loader.State()
	err := loader.AddRaw("foo", nil)
	if err == nil {
		t.Error("expected error on mutating consumed loader")
	}
}
