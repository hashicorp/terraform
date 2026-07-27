// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
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
	ListBlockAddress string                     `json:"list_block_address"`
	OverallResult    queryPolicyResult          `json:"overall_result"`
	Results          []queryPolicyIdentityResult `json:"results"`
	PassedPolicies   []viewjson.PolicyMetadata  `json:"passed_policies"`
}

type queryPolicyIdentityResult struct {
	Identity      map[string]string          `json:"identity,omitempty"`
	TargetAddress string                     `json:"target_address"`
	Result        queryPolicyResult          `json:"result"`
	Policies      []queryPolicyPolicyResult  `json:"policies"`
}

type queryPolicyPolicyResult struct {
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
		addr := queryPolicyListBlockAddr(diag.Description().Detail)
		if addr == "" {
			continue
		}
		block := v.block(addr)
		block.WarningDiags = append(block.WarningDiags, diag)
	}
}

func (v *queryPolicyView) AddResult(addr string, resp policy.EvaluationResponse) bool {
	if resp.ListBlockAddr == "" {
		return false
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	block := v.block(resp.ListBlockAddr)
	identityResult := block.ResultsByTarget[addr]
	if identityResult == nil {
		identityResult = &queryPolicyIdentityResult{
			Identity:      copyIdentity(resp.Identity),
			TargetAddress: addr,
			Policies:      make([]queryPolicyPolicyResult, 0, len(resp.Policies)),
		}
		block.ResultsByTarget[addr] = identityResult
	} else if identityResult.Identity == nil {
		identityResult.Identity = copyIdentity(resp.Identity)
	}

	identityResult.Result = queryPolicyResultFromEvaluation(resp.Overall)
	identityResult.Policies = identityResult.Policies[:0]

	diagsByPolicy := make(map[string][]viewjson.Diagnostic, len(resp.Policies))
	unmatchedDiags := make([]viewjson.Diagnostic, 0)
	for _, diag := range resp.Diagnostics {
		jsonDiag := *viewjson.NewDiagnostic(diag, nil)
		extra := tfdiags.ExtraInfo[*policy.PolicyExtra](diag)
		if extra == nil || extra.Policy.Address == "" {
			unmatchedDiags = append(unmatchedDiags, jsonDiag)
			continue
		}
		diagsByPolicy[extra.Policy.Address] = append(diagsByPolicy[extra.Policy.Address], jsonDiag)
	}

	for _, pol := range resp.Policies {
		metadata := viewjson.MetadataFromPolicy(*pol)
		block.PolicyMetadata[pol.Address] = metadata
		policyDiags := diagsByPolicy[pol.Address]
		if len(policyDiags) == 0 && len(unmatchedDiags) > 0 && len(resp.Policies) == 1 {
			policyDiags = unmatchedDiags
		}
		identityResult.Policies = append(identityResult.Policies, queryPolicyPolicyResult{
			PolicyMetadata: metadata,
			Diagnostics:    policyDiags,
			Result:         queryPolicyResultFromEvaluation(pol.Result),
		})
	}

	return true
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
		policies := append([]queryPolicyPolicyResult(nil), result.Policies...)
		sort.Slice(policies, func(i, j int) bool {
			return policies[i].PolicyMetadata.PolicyName < policies[j].PolicyMetadata.PolicyName
		})
		results = append(results, queryPolicyIdentityResult{
			Identity:      copyIdentity(result.Identity),
			TargetAddress: result.TargetAddress,
			Result:        result.Result,
			Policies:      policies,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].TargetAddress < results[j].TargetAddress
	})

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

func queryPolicyListBlockAddr(detail string) string {
	const marker = "list block "
	idx := strings.Index(detail, marker)
	if idx == -1 {
		return ""
	}
	rest := detail[idx+len(marker):]
	if rest == "" {
		return ""
	}
	for i, r := range rest {
		if r == ' ' || r == ':' || r == '.' {
			return rest[:i]
		}
	}
	return rest
}

func copyIdentity(identity map[string]string) map[string]string {
	if len(identity) == 0 {
		return nil
	}
	ret := make(map[string]string, len(identity))
	for k, v := range identity {
		ret[k] = v
	}
	return ret
}

func renderQueryPolicySummaryHuman(summary queryPolicySummary) string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "Policy results for %s (%s)\n", summary.ListBlockAddress, strings.ToUpper(string(summary.OverallResult)))
	for _, result := range summary.Results {
		fmt.Fprintf(&buf, "  %s: %s\n", formatQueryPolicyIdentity(result.Identity), strings.ToUpper(string(result.Result)))
		for _, pol := range result.Policies {
			if pol.Result != queryPolicyResultFail {
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

