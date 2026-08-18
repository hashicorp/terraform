// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync"

	viewjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type queryPolicyResult string

const (
	queryPolicyResultPass    queryPolicyResult = "pass"
	queryPolicyResultFail    queryPolicyResult = "fail"
	queryPolicyResultUnknown queryPolicyResult = "unknown"
	queryPolicyResultError   queryPolicyResult = "error"
)

type queryPolicySummary struct {
	ListBlockAddress string                      `json:"list_block_address"`
	OverallResult    queryPolicyResult           `json:"overall_result"`
	Results          []queryPolicyIdentityResult `json:"results"`
	PassedPolicies   []viewjson.PolicyMetadata   `json:"passed_policies"`
}

type queryPolicyIdentityResult struct {
	Identity      map[string]string       `json:"identity,omitempty"`
	TargetAddress string                  `json:"target_address"`
	Result        queryPolicyResult       `json:"result"`
	Policies      []queryPolicyEvalResult `json:"policies"`
}

// queryPolicyEvalResult holds the per-policy outcome for a single evaluated
// identity within a list block.
type queryPolicyEvalResult struct {
	PolicyMetadata viewjson.PolicyMetadata `json:"policy_metadata"`
	Diagnostics    []viewjson.Diagnostic   `json:"diagnostics"`
	Result         queryPolicyResult       `json:"result"`
}

type queryPolicyBlock struct {
	ListBlockAddress string
	ResultsByTarget  map[string]*queryPolicyIdentityResult
	PolicyMetadata   map[string]viewjson.PolicyMetadata
	WarningDiags     tfdiags.Diagnostics
}

type queryPolicyView struct {
	mu     sync.Mutex
	blocks map[string]*queryPolicyBlock
}

func newQueryPolicyView() *queryPolicyView {
	return &queryPolicyView{
		blocks: make(map[string]*queryPolicyBlock),
	}
}

// AddWarningDiags routes warning diagnostics that carry a ListBlockAddrExtra
// into the matching queryPolicyBlock so they are emitted together with the
// policy summary at flush time. Diagnostics without that extra are silently
// ignored by this function; callers are responsible for emitting them via the
// normal diagnostic path.
//
// Only warning diagnostics are processed here. This is intentional: policy
// query warnings (e.g., "policy was skipped for this identity") belong in the
// query summary block output. Other severity levels are handled by standard
// diagnostic paths.
func (v *queryPolicyView) AddWarningDiags(diags tfdiags.Diagnostics) {
	if len(diags) == 0 {
		return
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	for _, diag := range diags {
		if diag.Severity() != tfdiags.Warning {
			continue
		}
		extra := tfdiags.ExtraInfo[*tfdiags.ListBlockAddrExtra](diag)
		if extra == nil || extra.ListBlockAddr == "" {
			continue
		}
		block := v.block(extra.ListBlockAddr)
		block.WarningDiags = append(block.WarningDiags, diag)
	}
}

func (v *queryPolicyView) AddResult(addr string, resp policy.EvaluationResponse) (bool, policy.Diagnostics) {
	if resp.ListBlockAddr == "" {
		return false, nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	block := v.block(resp.ListBlockAddr)
	identityResult := block.ResultsByTarget[addr]
	if identityResult == nil {
		identityResult = &queryPolicyIdentityResult{
			Identity:      maps.Clone(resp.Identity),
			TargetAddress: addr,
			Policies:      make([]queryPolicyEvalResult, 0, len(resp.Policies)),
		}
		block.ResultsByTarget[addr] = identityResult
	} else if identityResult.Identity == nil {
		identityResult.Identity = maps.Clone(resp.Identity)
	}

	identityResult.Result = queryPolicyResultFromEvaluation(resp.Overall)
	// Reset the per-policy slice so that a re-evaluation of the same
	// identity always reflects only the most-recent response.
	identityResult.Policies = identityResult.Policies[:0]

	diagPolicyAddrs := make([]string, len(resp.Diagnostics))
	for idx, diag := range resp.Diagnostics {
		extra := tfdiags.ExtraInfo[*policy.PolicyExtra](diag)
		if extra != nil {
			diagPolicyAddrs[idx] = extra.Policy.Address
		}
	}
	consumedDiags := make([]bool, len(resp.Diagnostics))

	for _, pol := range resp.Policies {
		metadata := viewjson.MetadataFromPolicy(*pol)
		block.PolicyMetadata[pol.Address] = metadata
		var policyDiags []viewjson.Diagnostic
		for idx, policyAddr := range diagPolicyAddrs {
			if consumedDiags[idx] || policyAddr != pol.Address {
				continue
			}
			policyDiags = append(policyDiags, *viewjson.NewDiagnostic(resp.Diagnostics[idx], nil))
			consumedDiags[idx] = true
		}
		identityResult.Policies = append(identityResult.Policies, queryPolicyEvalResult{
			PolicyMetadata: metadata,
			Diagnostics:    policyDiags,
			Result:         queryPolicyResultFromEvaluation(pol.Result),
		})
	}

	var unconsumed policy.Diagnostics
	for idx, diag := range resp.Diagnostics {
		if !consumedDiags[idx] {
			unconsumed = append(unconsumed, diag)
		}
	}
	return true, unconsumed
}

func (v *queryPolicyView) Flush(flush func(summary queryPolicySummary), warn func(tfdiags.Diagnostics)) {
	v.mu.Lock()
	blocks := make([]*queryPolicyBlock, 0, len(v.blocks))
	for _, block := range v.blocks {
		blocks = append(blocks, block)
	}
	v.blocks = make(map[string]*queryPolicyBlock)
	v.mu.Unlock()

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].ListBlockAddress < blocks[j].ListBlockAddress
	})

	for _, block := range blocks {
		if warn != nil && len(block.WarningDiags) > 0 {
			warn(block.WarningDiags)
		}
		flush(block.summary())
	}
}

func (v *queryPolicyView) HasResults() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.blocks) > 0
}

func (v *queryPolicyView) block(addr string) *queryPolicyBlock {
	block := v.blocks[addr]
	if block != nil {
		return block
	}
	block = &queryPolicyBlock{
		ListBlockAddress: addr,
		ResultsByTarget:  make(map[string]*queryPolicyIdentityResult),
		PolicyMetadata:   make(map[string]viewjson.PolicyMetadata),
	}
	v.blocks[addr] = block
	return block
}

func (b *queryPolicyBlock) summary() queryPolicySummary {
	results := make([]queryPolicyIdentityResult, 0, len(b.ResultsByTarget))
	for _, result := range b.ResultsByTarget {
		policies := append(make([]queryPolicyEvalResult, 0, len(result.Policies)), result.Policies...)
		sort.Slice(policies, func(i, j int) bool {
			return policies[i].PolicyMetadata.PolicyName < policies[j].PolicyMetadata.PolicyName
		})
		results = append(results, queryPolicyIdentityResult{
			Identity:      maps.Clone(result.Identity),
			TargetAddress: result.TargetAddress,
			Result:        result.Result,
			Policies:      policies,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].TargetAddress < results[j].TargetAddress
	})

	// passed_policies contains all policies evaluated against any identity in
	// this block, regardless of whether they passed or failed. The field name
	// is intentionally kept as "passed_policies" to match the RFC wire contract
	// (the UI uses it for column-header counts, not to filter by pass status).
	passedPolicies := make([]viewjson.PolicyMetadata, 0, len(b.PolicyMetadata))
	for _, metadata := range b.PolicyMetadata {
		passedPolicies = append(passedPolicies, metadata)
	}
	sort.Slice(passedPolicies, func(i, j int) bool {
		return passedPolicies[i].PolicyName < passedPolicies[j].PolicyName
	})

	return queryPolicySummary{
		ListBlockAddress: b.ListBlockAddress,
		OverallResult:    queryPolicySummaryOverall(results),
		Results:          results,
		PassedPolicies:   passedPolicies,
	}
}

func queryPolicySummaryOverall(results []queryPolicyIdentityResult) queryPolicyResult {
	overall := queryPolicyResultPass
	for _, result := range results {
		switch result.Result {
		case queryPolicyResultError:
			return queryPolicyResultError
		case queryPolicyResultFail:
			overall = queryPolicyResultFail
		case queryPolicyResultUnknown:
			if overall == queryPolicyResultPass {
				overall = queryPolicyResultUnknown
			}
		}
	}
	return overall
}

func queryPolicyResultFromEvaluation(result policy.EvaluateResult) queryPolicyResult {
	switch result {
	case policy.AllowResult:
		return queryPolicyResultPass
	case policy.DenyResult:
		return queryPolicyResultFail
	case policy.PolicyErrorResult, policy.SetupErrorResult:
		return queryPolicyResultError
	default:
		return queryPolicyResultUnknown
	}
}

func renderQueryPolicySummaryHuman(summary queryPolicySummary) string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "Policy results for %s (%s)\n", summary.ListBlockAddress, strings.ToUpper(string(summary.OverallResult)))
	for _, result := range summary.Results {
		label := strings.ToUpper(string(result.Result))
		if result.Result == queryPolicyResultPass && len(result.Policies) == 0 {
			label = "N/A"
		}
		fmt.Fprintf(&buf, "  %s: %s\n", formatQueryPolicyIdentity(result.Identity), label)
		for _, pol := range result.Policies {
			if pol.Result != queryPolicyResultFail && pol.Result != queryPolicyResultError {
				continue
			}
			for _, diag := range pol.Diagnostics {
				fmt.Fprintf(&buf, "    - %s: %s (%s)\n", pol.PolicyMetadata.PolicyName, diag.Summary, pol.PolicyMetadata.EnforcementLevel)
			}
		}
	}

	return strings.TrimRight(buf.String(), "\n")
}

func formatQueryPolicyIdentity(identity map[string]string) string {
	if len(identity) == 0 {
		return "<unknown identity>"
	}
	keys := make([]string, 0, len(identity))
	for key := range identity {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, identity[key]))
	}
	return strings.Join(parts, ", ")
}

func queryPolicyEvaluatedCount(view *queryPolicyView) int {
	if view == nil {
		return 0
	}
	view.mu.Lock()
	defer view.mu.Unlock()

	policies := make(map[string]struct{})
	for _, block := range view.blocks {
		for key := range block.PolicyMetadata {
			policies[key] = struct{}{}
		}
	}
	return len(policies)
}
