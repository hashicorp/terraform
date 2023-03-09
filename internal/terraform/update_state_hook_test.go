// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

func TestUpdateStateHook(t *testing.T) {
	mockHook := new(MockHook)

	state := states.NewState()
	state.Module(addrs.RootModuleInstance).SetOutputValue("foo", cty.StringVal("hello"), false)
	state.Module(addrs.RootModuleInstance)

	ctx := new(MockEvalContext)
	ctx.HookHook = mockHook
	ctx.StateState = state.SyncWrapper()

	if err := updateStateHook(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mockHook.PostStateUpdateCalled {
		t.Fatal("should call PostStateUpdate")
	}
	if os := mockHook.PostStateUpdateState.OutputValue(addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance)); os != nil && os.Value != cty.StringVal("hello") {
		t.Fatalf("wrong state passed to hook: %s", spew.Sdump(mockHook.PostStateUpdateState))
	}
}
