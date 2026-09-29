// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
)

func TestNodeApplyableDeferredInstance_decodeError(t *testing.T) {
	resourceAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_object",
		Name: "a",
	}
	instAddr := resourceAddr.Absolute(addrs.RootModuleInstance).Instance(addrs.NoKey)

	// This change can't be decoded, because the After value is not valid
	// msgpack for the schema.
	changeSrc := &plans.ResourceInstanceChangeSrc{
		Addr:        instAddr,
		PrevRunAddr: instAddr,
		ChangeSrc: plans.ChangeSrc{
			Action: plans.Create,
			Before: plans.DynamicValue([]byte{0xc0}),
			After:  plans.DynamicValue([]byte{0xff}),
		},
	}
	schema := &providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Computed: true},
			},
		},
	}

	newNode := func() *nodeApplyableDeferredInstance {
		node := &nodeApplyableDeferredInstance{
			NodeAbstractResourceInstance: NewNodeAbstractResourceInstance(instAddr),
			Reason:                       providers.DeferredReasonProviderConfigUnknown,
			ChangeSrc:                    changeSrc,
		}
		node.Schema = schema
		return node
	}

	tests := map[string]GraphNodeExecutable{
		"instance": newNode(),
		"partial": &nodeApplyableDeferredPartialInstance{
			nodeApplyableDeferredInstance: newNode(),
			PartialAddr:                   addrs.RootModuleInstance.UnexpandedResource(resourceAddr),
		},
	}

	for name, node := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &MockEvalContext{
				DeferralsState: deferring.NewDeferred(true),
			}

			diags := node.Execute(ctx, walkApply)
			if !diags.HasErrors() {
				t.Fatal("expected a decode error")
			}
			if got := len(ctx.DeferralsState.GetDeferredChanges()); got != 0 {
				t.Fatalf("expected no deferred changes to be reported, got %d", got)
			}
		})
	}
}
