// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	policyproto "github.com/hashicorp/terraform/internal/policy/proto"
)

// blockRelatesToOKBucket is a fake relationship policy that relates a block
// to the buckets its value's origins name, or, without origins, to the
// buckets whose id equals its value, like the engine does. It allows the
// block if some related bucket's value is "ok".
func (c *relationshipsTestClient) blockRelatesToOKBucket(runID, addr string) policy.EvaluateResult {
	block := relationshipsTestRecord(c.ReportInstancesRequests, runID, addr)
	if block == nil {
		return policy.PolicyErrorResult
	}
	value := relationshipsTestAttrs(block).GetAttr("value")
	var origins []string
	for _, ko := range block.Origins {
		if relationshipsTestPath(ko.KeyPath) != "value" {
			continue
		}
		for _, o := range ko.Origins {
			origins = append(origins, o.Address+"#"+relationshipsTestPath(o.Path))
		}
	}
	for _, req := range c.ReportInstancesRequests {
		if req.RunId != runID {
			continue
		}
		for _, r := range req.Records {
			if r.Type != "testing_resource" {
				continue
			}
			attrs := relationshipsTestAttrs(r)
			related := attrs.GetAttr("id").Equals(value).True()
			if len(origins) > 0 {
				related = slices.Contains(origins, r.Address+"#id")
			}
			if related && attrs.GetAttr("value").Equals(cty.StringVal("ok")).True() {
				return policy.AllowResult
			}
		}
	}
	return policy.DenyResult
}

func relationshipsTestRecord(reqs []*policyproto.ReportInstancesRequest, runID, addr string) *policyproto.InstanceRecord {
	for _, req := range reqs {
		if req.RunId != runID {
			continue
		}
		for _, r := range req.Records {
			if r.Address == addr {
				return r
			}
		}
	}
	return nil
}

func relationshipsTestAttrs(r *policyproto.InstanceRecord) cty.Value {
	val, err := msgpack.Unmarshal(r.Attrs.GetRaw(), cty.DynamicPseudoType)
	if err != nil {
		panic(err)
	}
	return val
}

func relationshipsTestPath(path *policyproto.AttributePath) string {
	var steps []string
	for _, step := range path.GetSteps() {
		switch sel := step.Selector.(type) {
		case *policyproto.AttributePath_Step_AttributeName:
			steps = append(steps, sel.AttributeName)
		case *policyproto.AttributePath_Step_ElementKeyInt:
			steps = append(steps, fmt.Sprintf("[%d]", sel.ElementKeyInt))
		case *policyproto.AttributePath_Step_ElementKeyString:
			steps = append(steps, fmt.Sprintf("[%q]", sel.ElementKeyString))
		}
	}
	return strings.Join(steps, ".")
}

// relationshipsTestOrigins returns the origins of a record's key paths as
// "keypath: address#path,..." or "keypath: no origin REASON,...".
func relationshipsTestOrigins(r *policyproto.InstanceRecord) []string {
	var ret []string
	for _, ko := range r.Origins {
		var parts []string
		for _, o := range ko.Origins {
			parts = append(parts, o.Address+"#"+relationshipsTestPath(o.Path))
		}
		for _, reason := range ko.NoOrigin {
			parts = append(parts, "no origin "+reason.String())
		}
		ret = append(ret, relationshipsTestPath(ko.KeyPath)+": "+strings.Join(parts, ","))
	}
	return ret
}

// A component input that refers directly to another component's output
// carries the origins of that output: the block in component.direct relates
// to the bucket its value comes from, testing_resource.bad, although the
// other bucket has the same id. An input that isn't a direct reference keeps
// no origin, VARIABLE, and relates by value.
func TestPlan_PolicyRelationshipsOriginsAcrossComponents(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)
	client.byOrigin = true

	got := planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-origins"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	begins := client.eventsWithPrefix("begin ")
	if len(begins) != 1 {
		t.Fatalf("want one run, got %q", begins)
	}
	runID := strings.Fields(begins[0])[1]

	tcs := []struct {
		component, address string
		origins            []string
		result             policy.EvaluateResult
	}{
		{
			component: "component.direct",
			address:   "component.direct.testing_deferred_resource.block",
			origins:   []string{"value: component.bucket.testing_resource.bad#id"},
			result:    policy.DenyResult,
		},
		{
			component: "component.computed",
			address:   "component.computed.testing_deferred_resource.block",
			origins:   []string{"value: no origin " + policyproto.NoOriginReason_VARIABLE_NO_ORIGIN_REASON.String()},
			result:    policy.AllowResult,
		},
		{
			component: "component.chained",
			address:   "component.chained.testing_deferred_resource.block",
			origins:   []string{"value: component.bucket.testing_resource.good#id"},
			result:    policy.AllowResult,
		},
		{
			component: `component.keyed["a"]`,
			address:   `component.keyed["a"].testing_deferred_resource.block`,
			origins:   []string{`value: component.keyed_bucket["a"].testing_resource.good#id`},
			result:    policy.AllowResult,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.component, func(t *testing.T) {
			record := relationshipsTestRecord(client.ReportInstancesRequests, runID, tc.address)
			if record == nil {
				t.Fatalf("no record of %s", tc.address)
			}
			if got := relationshipsTestOrigins(record); !slices.Equal(got, tc.origins) {
				t.Errorf("wrong origins\ngot:  %q\nwant: %q", got, tc.origins)
			}
			if res := got[tc.component]["testing_deferred_resource.block"]; res.Overall != tc.result {
				t.Errorf("wrong result: got %s, want %s", res.Overall, tc.result)
			}
		})
	}
}
