// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Apply_savedPlanProviderSourceDrift(t *testing.T) {
	t.Parallel()

	const providerSourceLocal = `locals {
  provider_source = trimspace(file("${path.module}/provider.txt"))
}`

	tests := []struct {
		name    string
		source  string
		locals  string
		changed bool
	}{
		{
			name:   "direct_file/unchanged",
			source: "trimspace(file(%q))",
		},
		{
			name:    "direct_file/changed",
			source:  "trimspace(file(%q))",
			changed: true,
		},
		{
			name:   "local_file/unchanged",
			source: "local.provider_source",
			locals: providerSourceLocal,
		},
		{
			name:    "local_file/changed",
			source:  "local.provider_source",
			locals:  providerSourceLocal,
			changed: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			plannedProvider := addrs.MustParseProviderSourceString("example.com/acme/test")
			otherProvider := addrs.MustParseProviderSourceString("example.com/beta/test")
			dir := t.TempDir()
			providerFile := filepath.Join(dir, "provider.txt")
			if err := os.WriteFile(providerFile, []byte(plannedProvider.String()+"\n"), 0600); err != nil {
				t.Fatal(err)
			}

			source := test.source
			if test.locals == "" {
				source = fmt.Sprintf(source, filepath.ToSlash(providerFile))
			}
			configSource := fmt.Sprintf(`
terraform {
  required_providers {
    test = { source = %s }
    acme = { source = "example.com/acme/test" }
    beta = { source = "example.com/beta/test" }
  }
}

%s

resource "test_instance" "example" {
  foo = "bar"
}
`, source, test.locals)
			if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(configSource), 0600); err != nil {
				t.Fatal(err)
			}

			loader, cleanup := configload.NewLoaderForTests(t)
			defer cleanup()
			config, snap, diags := testLoadWithSnapshot(dir, loader, nil)
			tfdiags.AssertNoErrors(t, diags)

			// Both possible sources are valid locked dependencies, so lock
			// validation alone cannot detect a changed resource binding.
			locks := depsfile.NewLocks()
			for _, addr := range []addrs.Provider{plannedProvider, otherProvider} {
				locks.SetProvider(addr, providerreqs.MustParseVersion("1.0.0"), nil, nil)
			}
			if errs := config.VerifyDependencySelections(locks); len(errs) != 0 {
				t.Fatalf("invalid planning dependency selections: %s", errs)
			}

			planned := testProvider("test")
			planned.PlanResourceChangeFn = testDiffFn
			planned.ApplyResourceChangeFn = testApplyFn
			other := testProvider("test")
			other.PlanResourceChangeFn = testDiffFn
			other.ApplyResourceChangeFn = testApplyFn
			factories := map[addrs.Provider]providers.Factory{
				plannedProvider: testProviderFuncFixed(planned),
				otherProvider:   testProviderFuncFixed(other),
			}
			ctx := testContext2(t, &ContextOpts{Providers: factories})
			plan, diags := ctx.Plan(config, states.NewState(), DefaultPlanOpts)
			tfdiags.AssertNoErrors(t, diags)

			addr := mustResourceInstanceAddr("test_instance.example")
			change := plan.Changes.ResourceInstance(addr)
			if change == nil {
				t.Fatal("missing planned change for test_instance.example")
			}
			if got := change.Action; got != plans.Create {
				t.Fatalf("planned action = %s, want %s", got, plans.Create)
			}
			if got := change.ProviderAddr.Provider; !got.Equals(plannedProvider) {
				t.Fatalf("planned provider = %s, want %s", got, plannedProvider)
			}

			// Change only the external input before rebuilding configuration
			// from the saved plan's snapshot, as the CLI does during apply.
			if test.changed {
				if err := os.WriteFile(providerFile, []byte(otherProvider.String()+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctxOpts, applyConfig, savedPlan, err := contextOptsForPlanViaFile(t, snap, plan)
			if err != nil {
				t.Fatalf("failed to round-trip through planfile: %s", err)
			}
			if errs := applyConfig.VerifyDependencySelections(locks); len(errs) != 0 {
				t.Fatalf("invalid apply dependency selections: %s", errs)
			}
			wantProvider := plannedProvider
			if test.changed {
				wantProvider = otherProvider
			}
			if got := applyConfig.Module.ProviderRequirements.RequiredProviders["test"].Type; !got.Equals(wantProvider) {
				t.Fatalf("wrong rebuilt provider: got %s, want %s", got, wantProvider)
			}
			ctxOpts.Providers = factories
			ctx = testContext2(t, ctxOpts)
			state, diags := ctx.Apply(savedPlan, applyConfig, nil)

			if !test.changed {
				tfdiags.AssertNoErrors(t, diags)
				if !planned.ApplyResourceChangeCalled {
					t.Error("planned provider's ApplyResourceChange was not called")
				}
				if other.ApplyResourceChangeCalled {
					t.Error("unplanned provider's ApplyResourceChange was called")
				}
				if state == nil || state.ResourceInstance(addr) == nil {
					t.Fatal("unchanged saved plan did not create the resource")
				}
				if got := state.Resource(addr.ContainingResource()).ProviderConfig.Provider; !got.Equals(plannedProvider) {
					t.Errorf("wrong provider in state: got %s, want %s", got, plannedProvider)
				}
				return
			}

			if !diags.HasErrors() {
				t.Error("saved plan apply must reject a changed provider source")
			} else {
				// A late filesystem-function inconsistency is insufficient:
				// explain the provider mismatch before applying any changes.
				var foundMismatch bool
				for _, diag := range diags {
					if diag.Severity() != tfdiags.Error {
						continue
					}
					desc := diag.Description()
					text := desc.Summary + "\n" + desc.Detail
					if strings.Contains(text, addr.String()) &&
						strings.Contains(text, plannedProvider.String()) &&
						strings.Contains(text, otherProvider.String()) {
						foundMismatch = true
						break
					}
				}
				if !foundMismatch {
					t.Errorf("expected diagnostic identifying resource and provider mismatch, got: %s", diags.Err())
				}
			}
			if planned.ApplyResourceChangeCalled {
				t.Error("planned provider's ApplyResourceChange was called despite provider drift")
			}
			if other.ApplyResourceChangeCalled {
				t.Error("unplanned provider's ApplyResourceChange was called despite provider drift")
			}
			if state != nil && state.ResourceInstance(addr) != nil {
				t.Errorf("provider drift must not create a resource in state: %s", state)
			}
		})
	}
}

func TestContext2Apply_savedPlanActionProviderSourceDrift(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		changed bool
	}{
		{name: "unchanged"},
		{name: "changed", changed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			plannedProvider := addrs.MustParseProviderSourceString("example.com/acme/action")
			otherProvider := addrs.MustParseProviderSourceString("example.com/beta/action")
			dir := t.TempDir()
			providerFile := filepath.Join(dir, "provider.txt")
			if err := os.WriteFile(providerFile, []byte(plannedProvider.String()+"\n"), 0600); err != nil {
				t.Fatal(err)
			}

			// Only the action's provider source is dynamic, so the triggering
			// resource keeps its provider and only the action can drift.
			configSource := fmt.Sprintf(`
terraform {
  required_providers {
    action = { source = trimspace(file(%q)) }
    acme   = { source = "example.com/acme/action" }
    beta   = { source = "example.com/beta/action" }
  }
}

action "action_example" "hello" {}

resource "test_object" "a" {
  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.action_example.hello]
    }
  }
}
`, filepath.ToSlash(providerFile))
			if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(configSource), 0600); err != nil {
				t.Fatal(err)
			}

			loader, cleanup := configload.NewLoaderForTests(t)
			defer cleanup()
			config, snap, diags := testLoadWithSnapshot(dir, loader, nil)
			tfdiags.AssertNoErrors(t, diags)

			invokeActionFn := func(providers.InvokeActionRequest) providers.InvokeActionResponse {
				return providers.InvokeActionResponse{
					Events: func(yield func(providers.InvokeActionEvent) bool) {
						yield(providers.InvokeActionEvent_Completed{})
					},
				}
			}
			resourceProvider := simpleMockProvider()
			planned := testContextActionProvider(invokeActionFn)
			other := testContextActionProvider(invokeActionFn)
			factories := map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(resourceProvider),
				plannedProvider:                  testProviderFuncFixed(planned),
				otherProvider:                    testProviderFuncFixed(other),
			}
			ctx := testContext2(t, &ContextOpts{Providers: factories})
			plan, diags := ctx.Plan(config, states.NewState(), DefaultPlanOpts)
			tfdiags.AssertNoErrors(t, diags)

			if got := len(plan.Changes.ActionInvocations); got != 1 {
				t.Fatalf("planned %d action invocations, want 1", got)
			}
			invocation := plan.Changes.ActionInvocations[0]
			actionAddr := invocation.Addr
			if got := invocation.ProviderAddr.Provider; !got.Equals(plannedProvider) {
				t.Fatalf("planned action provider = %s, want %s", got, plannedProvider)
			}

			// Change only the action's external input before rebuilding
			// configuration from the saved plan's snapshot.
			if test.changed {
				if err := os.WriteFile(providerFile, []byte(otherProvider.String()+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctxOpts, applyConfig, savedPlan, err := contextOptsForPlanViaFile(t, snap, plan)
			if err != nil {
				t.Fatalf("failed to round-trip through planfile: %s", err)
			}
			ctxOpts.Providers = factories
			ctx = testContext2(t, ctxOpts)
			state, diags := ctx.Apply(savedPlan, applyConfig, nil)

			resourceAddr := mustResourceInstanceAddr("test_object.a")
			if !test.changed {
				tfdiags.AssertNoErrors(t, diags)
				if !planned.InvokeActionCalled {
					t.Error("planned provider's InvokeAction was not called")
				}
				if other.InvokeActionCalled {
					t.Error("unplanned provider's InvokeAction was called")
				}
				if state == nil || state.ResourceInstance(resourceAddr) == nil {
					t.Error("unchanged saved plan did not create the triggering resource")
				}
				return
			}

			if !diags.HasErrors() {
				t.Error("saved plan apply must reject a changed action provider source")
			} else {
				var foundMismatch bool
				for _, diag := range diags {
					if diag.Severity() != tfdiags.Error {
						continue
					}
					desc := diag.Description()
					text := desc.Summary + "\n" + desc.Detail
					if strings.Contains(text, actionAddr.String()) &&
						strings.Contains(text, plannedProvider.String()) &&
						strings.Contains(text, otherProvider.String()) {
						foundMismatch = true
						break
					}
				}
				if !foundMismatch {
					t.Errorf("expected diagnostic identifying action and provider mismatch, got: %s", diags.Err())
				}
			}
			if planned.InvokeActionCalled {
				t.Error("planned provider's InvokeAction was called despite provider drift")
			}
			if other.InvokeActionCalled {
				t.Error("unplanned provider's InvokeAction was called despite provider drift")
			}
			if resourceProvider.ApplyResourceChangeCalled {
				t.Error("ApplyResourceChange was called despite provider drift")
			}
			if state != nil && state.ResourceInstance(resourceAddr) != nil {
				t.Errorf("provider drift must not create a resource in state: %s", state)
			}
		})
	}
}
