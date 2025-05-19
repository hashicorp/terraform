// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	providertest "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
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
							Body: &configschema.Block{},
						},
						ResourceTypes: map[string]providers.Schema{
							"happycloud_thingy": providers.Schema{
								Body: &configschema.Block{},
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
}
