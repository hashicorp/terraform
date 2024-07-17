// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	providertest "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestNamedPromisesPlan(t *testing.T) {
	// The goal of this test is to make sure we retain namedPromiseReporter
	// coverage over various important object types, so that we don't
	// accidentally regress the quality of self-reference ("dependency cycle")
	// errors under future maintenence.
	//
	// It isn't totally comprehensive over all implementations of
	// namedPromiseReporter, but we do aim to cover the main cases that a
	// typical stack configuration might hit.
	//
	// This is intentionally a test of the namedPromiseReporter implementations
	// directly, rather than of the dependency-message-building logic built
	// in terms of it, because the goal is for namedPromiseReporter to return
	// everything and then the diagnostic reporter to cherry-pick only the
	// subset of names it needs, and because this way we can get more test
	// coverage without needing fixtures for every possible combination of
	// self-references.

	cfg := testStackConfig(t, "planning", "named_promises")

	providerAddrs := addrs.MustParseProviderSourceString("example.com/test/happycloud")
	lock := depsfile.NewLocks()
	lock.SetProvider(
		providerAddrs,
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)

	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode: plans.NormalMode,
		InputVariableValues: map[stackaddrs.InputVariable]ExternalInputValue{
			{Name: "in"}: ExternalInputValue{
				Value: cty.StringVal("hello"),
			},
		},
		ProviderFactories: ProviderFactories{
			providerAddrs: providers.FactoryFixed(
				&providertest.MockProvider{
					GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
						Provider: providers.Schema{
							Block: &configschema.Block{},
						},
						ResourceTypes: map[string]providers.Schema{
							"happycloud_thingy": providers.Schema{
								Block: &configschema.Block{},
							},
						},
					},
				},
			),
		},
		DependencyLocks: *lock,
		PlanTimestamp:   time.Now().UTC(),
	})

	// We don't actually really care about the plan here. We just want the
	// side-effect of getting a bunch of promises created inside "main", which
	// we'll then ask about below.
	_, diags := testPlan(t, main)
	assertNoDiagnostics(t, diags)

	wantNames := collections.NewSetCmp[string](
		// Component-related
		`component.foo`,
		`component.foo modules`,
		`component.foo for_each`,
		`component.foo instances`,

		// Nested-stack-related
		`stack.child collected outputs`,
		`stack.child inputs`,
		`stack.child for_each`,
		`stack.child instances`,

		// Provider-related
		`example.com/test/happycloud schema`,
		`provider["example.com/test/happycloud"].main`,
		`provider["example.com/test/happycloud"].main for_each`,
		`provider["example.com/test/happycloud"].main instances`,

		// Output-value-related
		`output.out value`,
		`stack.child.output.out value`,
		`output.out`,
		`stack.child.output.out`,

		// Input-variable-related
		`var.in`,
		`stack.child.var.in`,
	)
	gotNames := collections.NewSetCmp[string]()
	ids := map[string]promising.PromiseID{}
	main.reportNamedPromises(func(id promising.PromiseID, name string) {
		gotNames.Add(name)
		// We'll also remember the id associated with each name so that
		// we can test the diagnostic message rendering below.
		ids[name] = id
		// NOTE: Some of the names get reused across both a config object
		// and its associated dynamic object when there are no dynamic
		// instance keys involved, and for those it's unspecified which
		// promise ID will "win", but that's fine for our purposes here
		// because we're only testing that some specific names get
		// included into the error messages and so it doesn't matter which
		// of the promise ids we use to achieve that.
	})

	if diff := cmp.Diff(wantNames, gotNames, collections.CmpOptions); diff != "" {
		// If you're here because you've seen a failure where some of the
		// wanted names seem to have vanished, and you weren't intentionally
		// trying to remove them, check to make sure that the type that was
		// supposed to report that name is still reachable indirectly from the
		// Main.reportNamedPromises implementation.
		t.Errorf("wrong promise names\n%s", diff)
	}

	// Since we're now holding all of the information required, let's also
	// test that we can render some self-dependency and resolution failure
	// diagnostic messages.
	t.Run("diagnostics", func(t *testing.T) {
		// For this we need to choose some specific promise ids to report.
		// It doesn't matter which ones we use but we can only proceed if
		// they were ones detected by the reportNamedPromises call earlier.
		providerSchemaPromise := ids[`example.com/test/happycloud schema`]
		stackCallInstancesPromise := ids[`stack.child instances`]
		if providerSchemaPromise == promising.NoPromise || stackCallInstancesPromise == promising.NoPromise {
			t.Fatalf("don't have the promise ids required to test diagnostic rendering")
		}

		t.Run("just one self-reference", func(t *testing.T) {
			err := promising.ErrSelfDependent{stackCallInstancesPromise}
			diag := taskSelfDependencyDiagnostic{
				err:  err,
				root: main,
			}
			got := diag.Description()
			want := tfdiags.Description{
				Summary: `Self-dependent item in configuration`,
				Detail:  `The item "stack.child instances" depends on its own results, so there is no correct order of operations.`,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong diagnostic description\n%s", diff)
			}
		})
		t.Run("multiple self-references", func(t *testing.T) {
			err := promising.ErrSelfDependent{
				providerSchemaPromise,
				stackCallInstancesPromise,
			}
			diag := taskSelfDependencyDiagnostic{
				err:  err,
				root: main,
			}
			got := diag.Description()
			want := tfdiags.Description{
				Summary: `Self-dependent items in configuration`,
				Detail: `The following items in your configuration form a circular dependency chain through their references:
  - example.com/test/happycloud schema
  - stack.child instances

Terraform uses references to decide a suitable order for performing operations, so configuration items may not refer to their own results either directly or indirectly.`,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong diagnostic description\n%s", diff)
			}
		})
		t.Run("just one failure to resolve", func(t *testing.T) {
			err := promising.ErrUnresolved{stackCallInstancesPromise}
			diag := taskPromisesUnresolvedDiagnostic{
				err:  err,
				root: main,
			}
			got := diag.Description()
			want := tfdiags.Description{
				Summary: `Stack language evaluation error`,
				Detail: `While evaluating the stack configuration, the following items were left unresolved:
  - stack.child instances

Other errors returned along with this one may provide more details. This is a bug in Teraform; please report it!`,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong diagnostic description\n%s", diff)
			}
		})
		t.Run("multiple failures to resolve", func(t *testing.T) {
			err := promising.ErrUnresolved{
				providerSchemaPromise,
				stackCallInstancesPromise,
			}
			diag := taskPromisesUnresolvedDiagnostic{
				err:  err,
				root: main,
			}
			got := diag.Description()
			want := tfdiags.Description{
				Summary: `Stack language evaluation error`,
				Detail: `While evaluating the stack configuration, the following items were left unresolved:
  - example.com/test/happycloud schema
  - stack.child instances

Other errors returned along with this one may provide more details. This is a bug in Teraform; please report it!`,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong diagnostic description\n%s", diff)
			}
		})
	})
}
