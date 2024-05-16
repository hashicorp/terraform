// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func init() {
	// Initialize the backends
	backendInit.Init(nil)
}

func TestMoveResourceState_DataStore(t *testing.T) {
	t.Parallel()

	nullResourceStateValue := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test"),
	})
	nullResourceStateJSON, err := ctyjson.Marshal(nullResourceStateValue, nullResourceStateValue.Type())

	if err != nil {
		t.Fatalf("failed to marshal null resource state: %s", err)
	}

	provider := &Provider{}
	req := providers.MoveResourceStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/null",
		SourceStateJSON:       nullResourceStateJSON,
		SourceTypeName:        "null_resource",
		TargetTypeName:        "terraform_data",
	}
	resp := provider.MoveResourceState(req)

	if resp.Diagnostics.HasErrors() {
		t.Errorf("unexpected diagnostics: %s", resp.Diagnostics.Err())
	}

	expectedTargetState := cty.ObjectVal(map[string]cty.Value{
		"id":               cty.StringVal("test"),
		"input":            cty.NullVal(cty.DynamicPseudoType),
		"output":           cty.NullVal(cty.DynamicPseudoType),
		"triggers_replace": cty.NullVal(cty.DynamicPseudoType),
	})

	if !resp.TargetState.RawEquals(expectedTargetState) {
		t.Errorf("expected state was:\n%#v\ngot state is:\n%#v\n", expectedTargetState, resp.TargetState)
	}
}

func TestMoveResourceState_NonExistentResource(t *testing.T) {
	t.Parallel()

	provider := &Provider{}
	req := providers.MoveResourceStateRequest{
		TargetTypeName: "nonexistent_resource",
	}
	resp := provider.MoveResourceState(req)

	if !resp.Diagnostics.HasErrors() {
		t.Fatal("expected diagnostics")
	}
}
