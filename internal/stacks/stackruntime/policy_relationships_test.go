// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	policyproto "github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
)

// relationshipsTestClient is a policy client with relationships whose fake
// relationship policy requires every testing_deferred_resource (the "block")
// to have a testing_resource (the "bucket") in the same run. Its
// policies without relationships call getresources for testing_resource.
type relationshipsTestClient struct {
	*policy.MockClient

	mu sync.Mutex
	// events are the client's calls in order.
	events []string
	// getResources counts the testing_resource objects getresources returned
	// to an evaluation, by its target type.
	getResources map[string]int

	// failStatuses makes the ReportInstances that carries the statuses
	// fail with a transport error.
	failStatuses bool
	// rejected lists the runs whose statuses failed to report.
	rejected map[string]bool
}

const relationshipsTestProvider = "registry.terraform.io/hashicorp/testing"

func newRelationshipsTestClient(t *testing.T) *relationshipsTestClient {
	t.Helper()
	c := &relationshipsTestClient{
		MockClient:   policy.NewTestMockClient(t),
		getResources: make(map[string]int),
		rejected:     make(map[string]bool),
	}
	c.RelationshipsSupportedResponse = true
	c.BeginRunFn = func(_ context.Context, req *policyproto.BeginRunRequest) (*policyproto.BeginRunResponse, error) {
		c.log("begin %s %s %s", req.RunId, req.Runtime, req.Stage)
		return &policyproto.BeginRunResponse{Spec: &policyproto.CollectionSpec{Types: []*policyproto.TypeSpec{
			{ProviderSource: relationshipsTestProvider, Type: "testing_resource"},
			{ProviderSource: relationshipsTestProvider, Type: "testing_deferred_resource"},
		}}}, nil
	}
	c.ReportInstancesFn = func(_ context.Context, req *policyproto.ReportInstancesRequest) (*policyproto.ReportInstancesResponse, error) {
		var addrs []string
		for _, r := range req.Records {
			addrs = append(addrs, r.Address)
		}
		var statuses []string
		for _, s := range req.Statuses {
			statuses = append(statuses, s.Type+"="+s.Completeness.String())
		}
		c.log("report %s records=%s statuses=%s", req.RunId, strings.Join(addrs, ","), strings.Join(statuses, ","))
		if c.failStatuses && len(req.Statuses) > 0 {
			c.mu.Lock()
			c.rejected[req.RunId] = true
			c.mu.Unlock()
			return nil, fmt.Errorf("connection lost")
		}
		return &policyproto.ReportInstancesResponse{}, nil
	}
	c.FinishRunFn = func(_ context.Context, req *policyproto.FinishRunRequest) (*policyproto.FinishRunResponse, error) {
		c.log("finish %s aborted=%t", req.RunId, req.Aborted)
		return &policyproto.FinishRunResponse{}, nil
	}
	c.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*policyproto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		c.log("evaluate %s %s %s %s", req.Selection, req.RunID, req.Target, req.Meta.GetAddress())
		var resp policy.EvaluationResponse
		if req.Selection != policyproto.PolicySelection_RELATIONSHIP_POLICY_SELECTION {
			resources, _, err := req.Callbacks.GetResources(ctx, "testing_resource", cty.NullVal(cty.DynamicPseudoType))
			if err != nil {
				t.Errorf("getresources failed: %s", err)
			}
			c.mu.Lock()
			c.getResources[req.Target] = len(resources)
			c.mu.Unlock()
			resp = relationshipsTestResponse(policy.AllowResult, "plain")
		}
		if req.Selection != policyproto.PolicySelection_NON_RELATIONSHIP_POLICY_SELECTION && req.Target == "testing_deferred_resource" {
			resp = resp.Merge(relationshipsTestResponse(c.blockHasBucket(req.RunID), "block has a bucket"))
		}
		return resp
	}
	return c
}

func (c *relationshipsTestClient) log(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, fmt.Sprintf(format, args...))
}

// blockHasBucket evaluates the relationship policy against the run's
// reports. EvaluateFn holds the mock's lock, so it can read them.
func (c *relationshipsTestClient) blockHasBucket(runID string) policy.EvaluateResult {
	c.mu.Lock()
	rejected := c.rejected[runID]
	c.mu.Unlock()
	if rejected {
		// Like the engine, fail closed on a run whose data didn't arrive.
		return policy.PolicyErrorResult
	}
	hasBucket := false
	complete := false
	for _, req := range c.ReportInstancesRequests {
		if req.RunId != runID {
			continue
		}
		for _, r := range req.Records {
			hasBucket = hasBucket || r.Type == "testing_resource"
		}
		for _, s := range req.Statuses {
			if s.Type == "testing_resource" {
				complete = s.Completeness == policyproto.TypeCompleteness_COMPLETE_TYPE_COMPLETENESS
			}
		}
	}
	switch {
	case !complete:
		return policy.UnknownResult
	case hasBucket:
		return policy.AllowResult
	default:
		return policy.DenyResult
	}
}

func relationshipsTestResponse(result policy.EvaluateResult, message string) policy.EvaluationResponse {
	return policy.EvaluationResponse{
		Overall:      result,
		Policies:     []*policy.Policy{mockPolicyObj(result)},
		Enforcements: []policy.EnforcementResult{{Result: result, Message: message}},
	}
}

func (c *relationshipsTestClient) eventsWithPrefix(prefix string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ret []string
	for _, e := range c.events {
		if strings.HasPrefix(e, prefix) {
			ret = append(ret, e)
		}
	}
	return ret
}

func (c *relationshipsTestClient) indexOf(event string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Index(c.events, event)
}

func relationshipsTestLocks() depsfile.Locks {
	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.1.0"),
		providerreqs.MustParseVersionConstraints("0.1.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)
	return *lock
}

func relationshipsTestProviders(t *testing.T) map[addrs.Provider]providers.Factory {
	return map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
			return stacks_testing_provider.NewProvider(t), nil
		},
	}
}

// checkDeploymentRun checks that the client saw exactly one STACKS run for
// the deployment: begun once, its statuses reported once after every
// component's records, the relationship policies evaluated only after that,
// and finished once. It returns the run's id.
func checkDeploymentRun(t *testing.T, c *relationshipsTestClient, stage policyproto.EvaluationStage) string {
	t.Helper()

	begins := c.eventsWithPrefix("begin ")
	if len(begins) != 1 {
		t.Fatalf("want one run for the deployment, got %d:\n%s", len(begins), strings.Join(c.events, "\n"))
	}
	runID := strings.Fields(begins[0])[1]
	if want := fmt.Sprintf("begin %s %s %s", runID, policyproto.RunRuntime_STACKS_RUN_RUNTIME, stage); begins[0] != want {
		t.Errorf("wrong BeginRun\ngot:  %s\nwant: %s", begins[0], want)
	}

	reports := c.eventsWithPrefix("report ")
	var withStatuses []string
	var records []string
	for _, r := range reports {
		if !strings.HasSuffix(r, "statuses=") {
			withStatuses = append(withStatuses, r)
		}
		if !strings.Contains(r, "records= ") {
			records = append(records, r)
		}
	}
	wantStatuses := fmt.Sprintf("report %s records= statuses=testing_resource=%s,testing_deferred_resource=%s", runID, policyproto.TypeCompleteness_COMPLETE_TYPE_COMPLETENESS, policyproto.TypeCompleteness_COMPLETE_TYPE_COMPLETENESS)
	if len(withStatuses) != 1 || withStatuses[0] != wantStatuses || reports[len(reports)-1] != wantStatuses {
		t.Fatalf("want the statuses once, in the last report:\n%s\ngot reports:\n%s", wantStatuses, strings.Join(reports, "\n"))
	}
	for _, addr := range []string{"component.bucket.testing_resource.bucket", "component.block.testing_deferred_resource.block"} {
		if !slices.ContainsFunc(records, func(r string) bool { return strings.Contains(r, addr) }) {
			t.Errorf("no record of %s; reports:\n%s", addr, strings.Join(reports, "\n"))
		}
	}

	statusesAt := c.indexOf(wantStatuses)
	relationshipEval := fmt.Sprintf("evaluate %s %s testing_deferred_resource component.block.testing_deferred_resource.block", policyproto.PolicySelection_RELATIONSHIP_POLICY_SELECTION, runID)
	if at := c.indexOf(relationshipEval); at < statusesAt {
		t.Errorf("want the relationship evaluation %q after the statuses, got:\n%s", relationshipEval, strings.Join(c.events, "\n"))
	}
	for _, e := range c.eventsWithPrefix("evaluate ") {
		if strings.HasPrefix(e, "evaluate "+policyproto.PolicySelection_NON_RELATIONSHIP_POLICY_SELECTION.String()+"  ") {
			// Policies without relationships: in the walk, without the run,
			// with the component's own address.
			if strings.Contains(e, "component.") {
				t.Errorf("evaluation without relationships has a component-qualified address: %s", e)
			}
			continue
		}
		if e != relationshipEval && !strings.HasPrefix(e, fmt.Sprintf("evaluate %s %s testing_resource component.bucket.", policyproto.PolicySelection_RELATIONSHIP_POLICY_SELECTION, runID)) {
			t.Errorf("unexpected evaluation: %s", e)
		}
	}

	finishes := c.eventsWithPrefix("finish ")
	if want := fmt.Sprintf("finish %s aborted=false", runID); len(finishes) != 1 || finishes[0] != want {
		t.Errorf("want FinishRun once: %s\ngot: %v", want, finishes)
	}
	return runID
}

func checkRelationshipResults(t *testing.T, got map[string]map[string]policy.EvaluationResponse) {
	t.Helper()
	block := got["component.block"]["testing_deferred_resource.block"]
	if block.Overall != policy.AllowResult {
		t.Errorf("block: want allow across components, got %s: %#v", block.Overall, block.Enforcements)
	}
	var messages []string
	for _, e := range block.Enforcements {
		messages = append(messages, e.Message)
	}
	if want := []string{"plain", "block has a bucket"}; !slices.Equal(messages, want) {
		t.Errorf("block: wrong enforcements: got %q, want %q", messages, want)
	}
	if bucket := got["component.bucket"]["testing_resource.bucket"]; bucket.Overall != policy.AllowResult || len(bucket.Enforcements) != 1 {
		t.Errorf("bucket: wrong result: %s %#v", bucket.Overall, bucket.Enforcements)
	}
}

// The block and the bucket are in different components. A run per component
// would report the bucket's type complete but absent to the block's
// relationship policy, which would fail; the deployment's run sees both.
func TestPlan_PolicyRelationshipsAcrossComponents(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)

	got := planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-cross-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	checkDeploymentRun(t, client, policyproto.EvaluationStage_PLAN_EVALUATION_STAGE)
	checkRelationshipResults(t, got)
}

func TestApply_PolicyRelationshipsAcrossComponents(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "policy-relationships-cross-component")

	plan := planForApplyTest(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            cfg,
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
	})

	client := newRelationshipsTestClient(t)
	got := applyAndCollectPolicyResults(t, ctx, ApplyRequest{
		Config:            cfg,
		Plan:              plan,
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	checkDeploymentRun(t, client, policyproto.EvaluationStage_APPLY_EVALUATION_STAGE)
	checkRelationshipResults(t, got)
}

// getresources in a policy without relationships stays scoped to the
// component being evaluated, as it was before the deployment's run.
func TestPlan_PolicyGetResourcesScopedToComponent(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-cross-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := map[string]int{
		"testing_resource":          1, // component.bucket sees its bucket
		"testing_deferred_resource": 0, // component.block doesn't see the bucket
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	for typ, n := range want {
		if got, ok := client.getResources[typ]; !ok || got != n {
			t.Errorf("getresources from %s: got %d (called: %t), want %d", typ, got, ok, n)
		}
	}
}

// A deferred instance in one component makes its type incomplete for the
// whole deployment, and the statuses only go out once every component has
// planned.
func TestPlan_PolicyRelationshipsDeferredComponent(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-cross-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			{Name: "block_deferred"}: {Value: cty.True},
		},
	})

	// The plan is done, so nothing calls the client anymore.
	reqs := client.ReportInstancesRequests
	var withStatuses []*policyproto.ReportInstancesRequest
	for _, req := range reqs {
		if len(req.Statuses) > 0 {
			withStatuses = append(withStatuses, req)
		}
	}
	if len(withStatuses) != 1 || withStatuses[0] != reqs[len(reqs)-1] {
		t.Fatalf("want the statuses once, in the last report; got %d reports with statuses of %d", len(withStatuses), len(reqs))
	}
	got := map[string]string{}
	for _, s := range withStatuses[0].Statuses {
		var deferred []string
		for _, d := range s.Deferred {
			deferred = append(deferred, d.Address)
		}
		got[s.Type] = s.Completeness.String() + " " + strings.Join(deferred, ",")
	}
	want := map[string]string{
		"testing_resource":          policyproto.TypeCompleteness_COMPLETE_TYPE_COMPLETENESS.String() + " ",
		"testing_deferred_resource": policyproto.TypeCompleteness_INCOMPLETE_DEFERRED_TYPE_COMPLETENESS.String() + " component.block.testing_deferred_resource.block",
	}
	for typ, w := range want {
		if got[typ] != w {
			t.Errorf("%s: got status %q, want %q", typ, got[typ], w)
		}
	}
}

// When the statuses fail to report, the relationship policies are still
// evaluated in the run, and fail closed, instead of giving no result.
func TestPlan_PolicyRelationshipsStatusesReportFails(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)
	client.failStatuses = true

	got := planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-cross-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	block, ok := got["component.block"]["testing_deferred_resource.block"]
	if !ok || block.Overall != policy.PolicyErrorResult {
		t.Errorf("block: want an error result, got %s (present: %t)", block.Overall, ok)
	}
	if !slices.ContainsFunc(block.Enforcements, func(e policy.EnforcementResult) bool { return e.Message == "block has a bucket" }) {
		t.Errorf("block: the relationship policy wasn't evaluated: %#v", block.Enforcements)
	}
	finishes := client.eventsWithPrefix("finish ")
	if len(finishes) != 1 || !strings.HasSuffix(finishes[0], "aborted=true") {
		t.Errorf("want FinishRun once, aborted: %v", finishes)
	}
}
