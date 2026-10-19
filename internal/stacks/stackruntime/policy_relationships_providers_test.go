// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"bytes"
	"context"
	"testing"

	"github.com/hashicorp/terraform/internal/plans"
	policyproto "github.com/hashicorp/terraform/internal/policy/proto"
)

// A stack provider configuration instance is one provider instance in the
// deployment's run, whichever components it's passed to, and its class is
// computed from its configuration with one key for the whole run, so
// configurations are comparable across components.
func TestPlan_PolicyRelationshipsProviderInstances(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-providers"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	providers := make(map[uint32]*policyproto.ProviderInstance)
	byComponent := make(map[string]*policyproto.ProviderInstance)
	for _, req := range client.ReportInstancesRequests {
		for _, p := range req.Providers {
			if _, ok := providers[p.Id]; ok {
				t.Errorf("provider instance %d reported twice", p.Id)
			}
			providers[p.Id] = p
		}
	}
	for _, req := range client.ReportInstancesRequests {
		for _, r := range req.Records {
			p := providers[r.ProviderInstanceId]
			if p == nil {
				t.Fatalf("%s: unknown provider instance %d", r.Address, r.ProviderInstanceId)
			}
			byComponent[r.Address] = p
		}
	}

	a1 := byComponent["component.a1.testing_resource.bucket"]
	a2 := byComponent["component.a2.testing_resource.bucket"]
	sameAsA := byComponent["component.same_as_a.testing_resource.bucket"]
	keyed := byComponent["component.keyed.testing_resource.bucket"]
	if a1 == nil || a2 == nil || sameAsA == nil || keyed == nil {
		t.Fatalf("missing records: %v", byComponent)
	}
	for _, tc := range []struct {
		p    *policyproto.ProviderInstance
		addr string
	}{
		{a1, "provider.testing.a"},
		{sameAsA, "provider.testing.same_as_a"},
		{keyed, `provider.testing.keyed["y"]`},
	} {
		if tc.p.ConfigAddress != tc.addr {
			t.Errorf("wrong config address: got %q, want %q", tc.p.ConfigAddress, tc.addr)
		}
		if !tc.p.Known || len(tc.p.ConfigClass) != 32 {
			t.Errorf("%s: want a known class, got known=%t class=%x", tc.addr, tc.p.Known, tc.p.ConfigClass)
		}
	}
	if a1 != a2 {
		t.Errorf("components sharing provider.testing.a have different provider instances: %v and %v", a1, a2)
	}
	if a1.Id == sameAsA.Id || !bytes.Equal(a1.ConfigClass, sameAsA.ConfigClass) {
		t.Errorf("want another instance with the same class: %v and %v", a1, sameAsA)
	}
	if bytes.Equal(a1.ConfigClass, keyed.ConfigClass) {
		t.Errorf("want different classes for different configurations: %v and %v", a1, keyed)
	}
}
