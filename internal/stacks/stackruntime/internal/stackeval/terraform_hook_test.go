// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
)

func TestTerraformHook(t *testing.T) {
	var gotRihd *hooks.ResourceInstanceStatusHookData
	testHooks := &Hooks{
		ReportResourceInstanceStatus: func(ctx context.Context, span any, rihd *hooks.ResourceInstanceStatusHookData) any {
			gotRihd = rihd
			return span
		},
	}
	componentAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance.Child("a", addrs.StringKey("boop")),
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{Name: "foo"},
			Key:       addrs.StringKey("beep"),
		},
	}

	makeHook := func() *componentInstanceTerraformHook {
		return &componentInstanceTerraformHook{
			ctx: context.Background(),
			seq: &hookSeq{
				tracking: "boop",
			},
			hooks: testHooks,
			addr:  componentAddr,
		}
	}

	resourceAddr := addrs.AbsResourceInstance{
		Module: addrs.RootModuleInstance,
		Resource: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			},
			Key: addrs.NoKey,
		},
	}
	providerAddr := addrs.Provider{
		Type:      "foo",
		Namespace: "hashicorp",
		Hostname:  "example.com",
	}
	resourceIdentity := terraform.HookResourceIdentity{
		Addr:         resourceAddr,
		ProviderAddr: providerAddr,
	}
	stackAddr := stackaddrs.AbsResourceInstanceObject{
		Component: componentAddr,
		Item:      resourceAddr.CurrentObject(),
	}

	t.Run("PreDiff", func(t *testing.T) {
		hook := makeHook()
		action, err := hook.PreDiff(resourceIdentity, addrs.NotDeposed, cty.NilVal, cty.NilVal)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if action != terraform.HookActionContinue {
			t.Errorf("wrong action: %#v", action)
		}
		if hook.seq.tracking != "boop" {
			t.Errorf("wrong tracking value: %#v", hook.seq.tracking)
		}

		wantRihd := &hooks.ResourceInstanceStatusHookData{
			Addr:         stackAddr,
			ProviderAddr: providerAddr,
			Status:       hooks.ResourceInstancePlanning,
		}
		if diff := cmp.Diff(gotRihd, wantRihd); diff != "" {
			t.Errorf("wrong status hook data:\n%s", diff)
		}
	})

	t.Run("PostDiff", func(t *testing.T) {
		hook := makeHook()
		action, err := hook.PostDiff(resourceIdentity, addrs.NotDeposed, plans.Create, cty.NilVal, cty.NilVal)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if action != terraform.HookActionContinue {
			t.Errorf("wrong action: %#v", action)
		}
		if hook.seq.tracking != "boop" {
			t.Errorf("wrong tracking value: %#v", hook.seq.tracking)
		}

		wantRihd := &hooks.ResourceInstanceStatusHookData{
			Addr:         stackAddr,
			ProviderAddr: providerAddr,
			Status:       hooks.ResourceInstancePlanned,
		}
		if diff := cmp.Diff(gotRihd, wantRihd); diff != "" {
			t.Errorf("wrong status hook data:\n%s", diff)
		}
	})

	t.Run("PreApply", func(t *testing.T) {
		hook := makeHook()
		action, err := hook.PreApply(resourceIdentity, addrs.NotDeposed, plans.Create, cty.NilVal, cty.NilVal)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if action != terraform.HookActionContinue {
			t.Errorf("wrong action: %#v", action)
		}
		if hook.seq.tracking != "boop" {
			t.Errorf("wrong tracking value: %#v", hook.seq.tracking)
		}

		wantRihd := &hooks.ResourceInstanceStatusHookData{
			Addr:         stackAddr,
			ProviderAddr: providerAddr,
			Status:       hooks.ResourceInstanceApplying,
		}
		if diff := cmp.Diff(gotRihd, wantRihd); diff != "" {
			t.Errorf("wrong status hook data:\n%s", diff)
		}
	})

	t.Run("PostApply", func(t *testing.T) {
		hook := makeHook()
		// It is invalid to call PostApply without first calling PreApply
		action, err := hook.PreApply(resourceIdentity, addrs.NotDeposed, plans.Create, cty.NilVal, cty.NilVal)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if action != terraform.HookActionContinue {
			t.Errorf("wrong action: %#v", action)
		}

		action, err = hook.PostApply(resourceIdentity, addrs.NotDeposed, cty.NilVal, nil)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if action != terraform.HookActionContinue {
			t.Errorf("wrong action: %#v", action)
		}
		if hook.seq.tracking != "boop" {
			t.Errorf("wrong tracking value: %#v", hook.seq.tracking)
		}

		wantRihd := &hooks.ResourceInstanceStatusHookData{
			Addr:         stackAddr,
			ProviderAddr: providerAddr,
			Status:       hooks.ResourceInstanceApplied,
		}
		if diff := cmp.Diff(gotRihd, wantRihd); diff != "" {
			t.Errorf("wrong status hook data:\n%s", diff)
		}
	})

	t.Run("PostApply errored", func(t *testing.T) {
		hook := makeHook()
		// It is invalid to call PostApply without first calling PreApply
		action, err := hook.PreApply(resourceIdentity, addrs.NotDeposed, plans.Create, cty.NilVal, cty.NilVal)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if action != terraform.HookActionContinue {
			t.Errorf("wrong action: %#v", action)
		}

		action, err = hook.PostApply(resourceIdentity, addrs.NotDeposed, cty.NilVal, errors.New("splines unreticulatable"))
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if action != terraform.HookActionContinue {
			t.Errorf("wrong action: %#v", action)
		}
		if hook.seq.tracking != "boop" {
			t.Errorf("wrong tracking value: %#v", hook.seq.tracking)
		}

		wantRihd := &hooks.ResourceInstanceStatusHookData{
			Addr:         stackAddr,
			ProviderAddr: providerAddr,
			Status:       hooks.ResourceInstanceErrored,
		}
		if diff := cmp.Diff(gotRihd, wantRihd); diff != "" {
			t.Errorf("wrong status hook data:\n%s", diff)
		}
	})

	t.Run("ResourceInstanceObjectAppliedAction", func(t *testing.T) {
		testCases := []struct {
			actions []plans.Action
			want    plans.Action
		}{
			{
				actions: []plans.Action{plans.NoOp},
				want:    plans.NoOp,
			},
			{
				actions: []plans.Action{plans.Create},
				want:    plans.Create,
			},
			{
				actions: []plans.Action{plans.Delete},
				want:    plans.Delete,
			},
			{
				actions: []plans.Action{plans.Update},
				want:    plans.Update,
			},
			{
				// We return a fallback of no-op if the object has no recorded
				// applied action.
				actions: []plans.Action{},
				want:    plans.NoOp,
			},
			{
				// Create-then-delete plans result in two separate apply
				// operations, which we need to recombine into a single one in
				// order to correctly count the operations.
				actions: []plans.Action{plans.Create, plans.Delete},
				want:    plans.CreateThenDelete,
			},
			{
				// See above: same for delete-then-create.
				actions: []plans.Action{plans.Delete, plans.Create},
				want:    plans.DeleteThenCreate,
			},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%v", tc.actions), func(t *testing.T) {
				hook := makeHook()

				for _, action := range tc.actions {
					_, err := hook.PreApply(resourceIdentity, addrs.NotDeposed, action, cty.NilVal, cty.NilVal)
					if err != nil {
						t.Fatalf("unexpected error in PreApply: %s", err)
					}

					_, err = hook.PostApply(resourceIdentity, addrs.NotDeposed, cty.NilVal, nil)
					if err != nil {
						t.Fatalf("unexpected error in PostApply: %s", err)
					}
				}

				got := hook.ResourceInstanceObjectAppliedAction(resourceAddr.CurrentObject())

				if got != tc.want {
					t.Errorf("wrong result: got %v, want %v", got, tc.want)
				}
			})
		}
	})
}
