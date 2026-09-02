// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"encoding/json"
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
	queryPolicyResultNA      queryPolicyResult = "n/a"
)

type PolicyQuerySummary struct {
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

// ParsePolicyQuerySummary parses and validates a policy_query_summary record.
func ParsePolicyQuerySummary(line []byte) (PolicyQuerySummary, error) {
	var summary PolicyQuerySummary
	if err := json.Unmarshal(line, &summary); err != nil {
		return PolicyQuerySummary{}, err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		return PolicyQuerySummary{}, err
	}
	for _, name := range []string{"results", "passed_policies"} {
		raw, ok := fields[name]
		trimmed := bytes.TrimSpace(raw)
		if !ok || len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '[' {
			return PolicyQuerySummary{}, fmt.Errorf("policy query summary %q must be an array", name)
		}
	}

	if strings.TrimSpace(summary.ListBlockAddress) == "" {
		return PolicyQuerySummary{}, fmt.Errorf("policy query summary has no list block address")
	}
	if !validQueryPolicyResult(summary.OverallResult) {
		return PolicyQuerySummary{}, fmt.Errorf("policy query summary has invalid overall result %q", summary.OverallResult)
	}
	for i, result := range summary.Results {
		if strings.TrimSpace(result.TargetAddress) == "" {
			return PolicyQuerySummary{}, fmt.Errorf("policy query summary result %d has no target address", i)
		}
		if !validQueryPolicyResult(result.Result) {
			return PolicyQuerySummary{}, fmt.Errorf("policy query summary result %d has invalid result %q", i, result.Result)
		}
		if result.Policies == nil {
			return PolicyQuerySummary{}, fmt.Errorf("policy query summary result %d has no policies array", i)
		}
		// An N/A row means no policies matched that resource; it must carry an
		// empty policies array. A non-empty array alongside "n/a" is malformed.
		if result.Result == queryPolicyResultNA && len(result.Policies) > 0 {
			return PolicyQuerySummary{}, fmt.Errorf("policy query summary result %d has result %q but non-empty policies array", i, result.Result)
		}
		for j, pol := range result.Policies {
			if strings.TrimSpace(pol.PolicyMetadata.PolicyName) == "" {
				return PolicyQuerySummary{}, fmt.Errorf("policy query summary result %d policy %d has no policy name", i, j)
			}
			if !validPerPolicyResult(pol.Result) {
				return PolicyQuerySummary{}, fmt.Errorf("policy query summary result %d policy %d has invalid result %q", i, j, pol.Result)
			}
		}
	}
	for i, metadata := range summary.PassedPolicies {
		if strings.TrimSpace(metadata.PolicyName) == "" {
			return PolicyQuerySummary{}, fmt.Errorf("policy query summary passed policy %d has no policy name", i)
		}
	}

	return summary, nil
}

// RenderPolicyQuerySummariesHuman renders a complete human-readable policy
// section for one or more list blocks.
func RenderPolicyQuerySummariesHuman(summaries []PolicyQuerySummary) string {
	if len(summaries) == 0 {
		return ""
	}

	ordered := append([]PolicyQuerySummary(nil), summaries...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].ListBlockAddress < ordered[j].ListBlockAddress
	})

	policies := make(map[queryPolicyKey]struct{})
	for _, summary := range ordered {
		for _, metadata := range summary.PassedPolicies {
			policies[queryPolicyKeyFromMetadata(metadata)] = struct{}{}
		}
		for _, result := range summary.Results {
			for _, pol := range result.Policies {
				policies[queryPolicyKeyFromMetadata(pol.PolicyMetadata)] = struct{}{}
			}
		}
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "Evaluated %d policies.", len(policies))
	for _, summary := range ordered {
		buf.WriteString("\n\n")
		buf.WriteString(renderQueryPolicySummaryHuman(summary))
	}
	return buf.String()
}

// validQueryPolicyResult returns true for any result value that is legal in an
// overall_result or per-identity result field. "n/a" is an intentional fifth
// status emitted by the producer when no policies matched a resource; it is
// valid in both positions but must not appear as a per-policy result.
func validQueryPolicyResult(result queryPolicyResult) bool {
	switch result {
	case queryPolicyResultPass, queryPolicyResultFail, queryPolicyResultUnknown, queryPolicyResultError, queryPolicyResultNA:
		return true
	default:
		return false
	}
}

// validPerPolicyResult returns true only for result values that are legal for
// an individual policy evaluation. "n/a" is not valid at this level.
func validPerPolicyResult(result queryPolicyResult) bool {
	switch result {
	case queryPolicyResultPass, queryPolicyResultFail, queryPolicyResultUnknown, queryPolicyResultError:
		return true
	default:
		return false
	}
}

type queryPolicyKey struct {
	Address       string
	PolicySetName string
	SourcePath    string
}

type queryPolicyBlock struct {
	ListBlockAddress string
	ResultsByTarget  map[string]*queryPolicyIdentityResult
	PolicyMetadata   map[queryPolicyKey]viewjson.PolicyMetadata
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
	if identityResult.Result == queryPolicyResultPass && len(resp.Policies) == 0 {
		identityResult.Result = queryPolicyResultNA
	}
	// Reset the per-policy slice so that a re-evaluation of the same
	// identity always reflects only the most-recent response.
	identityResult.Policies = identityResult.Policies[:0]

	jsonDiags := make([]viewjson.Diagnostic, len(resp.Diagnostics))
	diagPolicyKeys := make([]queryPolicyKey, len(resp.Diagnostics))
	consumedDiags := make([]bool, len(resp.Diagnostics))
	for idx, diag := range resp.Diagnostics {
		jsonDiags[idx] = *viewjson.NewDiagnostic(diag, nil)
		extra := tfdiags.ExtraInfo[*policy.PolicyExtra](diag)
		if extra != nil {
			diagPolicyKeys[idx] = queryPolicyKeyFromPolicy(extra.Policy)
		}
	}

	for _, pol := range resp.Policies {
		policyKey := queryPolicyKeyFromPolicy(*pol)
		metadata := viewjson.MetadataFromPolicy(*pol)
		block.PolicyMetadata[policyKey] = metadata
		var policyDiags []viewjson.Diagnostic
		for idx, diagPolicyKey := range diagPolicyKeys {
			if consumedDiags[idx] || diagPolicyKey.Address == "" || diagPolicyKey != policyKey {
				continue
			}
			policyDiags = append(policyDiags, jsonDiags[idx])
			consumedDiags[idx] = true
		}
		identityResult.Policies = append(identityResult.Policies, queryPolicyEvalResult{
			PolicyMetadata: metadata,
			Diagnostics:    policyDiags,
			Result:         queryPolicyResultFromEvaluation(pol.Result),
		})
	}

	unconsumed := make(policy.Diagnostics, 0)
	for idx, diag := range resp.Diagnostics {
		if !consumedDiags[idx] {
			unconsumed = append(unconsumed, diag)
		}
	}

	return true, unconsumed
}

func (v *queryPolicyView) Flush(flush func(summary PolicyQuerySummary)) {
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
		PolicyMetadata:   make(map[queryPolicyKey]viewjson.PolicyMetadata),
	}
	v.blocks[addr] = block
	return block
}

func (b *queryPolicyBlock) summary() PolicyQuerySummary {
	results := make([]queryPolicyIdentityResult, 0, len(b.ResultsByTarget))
	for _, result := range b.ResultsByTarget {
		policies := append(make([]queryPolicyEvalResult, 0, len(result.Policies)), result.Policies...)
		sort.Slice(policies, func(i, j int) bool {
			return queryPolicyMetadataLess(policies[i].PolicyMetadata, policies[j].PolicyMetadata)
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

	// passed_policies contains all distinct policies evaluated against any
	// identity in this block, as required by the query summary wire contract.
	passedPolicies := make([]viewjson.PolicyMetadata, 0, len(b.PolicyMetadata))
	for _, metadata := range b.PolicyMetadata {
		passedPolicies = append(passedPolicies, metadata)
	}
	sort.Slice(passedPolicies, func(i, j int) bool {
		return queryPolicyMetadataLess(passedPolicies[i], passedPolicies[j])
	})

	return PolicyQuerySummary{
		ListBlockAddress: b.ListBlockAddress,
		OverallResult:    queryPolicySummaryOverall(results),
		Results:          results,
		PassedPolicies:   passedPolicies,
	}
}

func queryPolicyKeyFromPolicy(pol policy.Policy) queryPolicyKey {
	return queryPolicyKey{
		Address:       pol.Address,
		PolicySetName: pol.PolicySetName,
		SourcePath:    pol.Directory,
	}
}

func queryPolicyKeyFromMetadata(metadata viewjson.PolicyMetadata) queryPolicyKey {
	return queryPolicyKey{
		Address:       metadata.PolicyName,
		PolicySetName: metadata.PolicySetName,
		SourcePath:    metadata.PolicySetPath,
	}
}

func queryPolicyMetadataLess(left, right viewjson.PolicyMetadata) bool {
	if left.PolicyName != right.PolicyName {
		return left.PolicyName < right.PolicyName
	}
	if left.PolicySetPath != right.PolicySetPath {
		return left.PolicySetPath < right.PolicySetPath
	}
	return left.PolicySetName < right.PolicySetName
}

func queryPolicySummaryOverall(results []queryPolicyIdentityResult) queryPolicyResult {
	if len(results) == 0 {
		// N/A describes an evaluated identity that matched no policies. With no
		// identities there was nothing to evaluate, so preserve the empty summary's
		// historical Pass result.
		return queryPolicyResultPass
	}

	overall := queryPolicyResultNA
	for _, result := range results {
		switch result.Result {
		case queryPolicyResultError:
			return queryPolicyResultError
		case queryPolicyResultFail:
			overall = queryPolicyResultFail
		case queryPolicyResultUnknown:
			if overall == queryPolicyResultNA || overall == queryPolicyResultPass {
				overall = queryPolicyResultUnknown
			}
		case queryPolicyResultPass:
			if overall == queryPolicyResultNA {
				overall = queryPolicyResultPass
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
	case policy.UnknownResult:
		return queryPolicyResultUnknown
	default:
		return queryPolicyResultError
	}
}

func renderQueryPolicySummaryHuman(summary PolicyQuerySummary) string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "Policy results for %s - %s\n", summary.ListBlockAddress, toPolicyResultLabel(summary.OverallResult))

	maxLen := 0
	for _, r := range summary.Results {
		if l := len(formatQueryPolicyIdentity(r.Identity)); l > maxLen {
			maxLen = l
		}
	}

	for _, result := range summary.Results {
		identity := formatQueryPolicyIdentity(result.Identity)
		label := toPolicyResultLabel(result.Result)
		if result.Result == queryPolicyResultPass && len(result.Policies) == 0 {
			label = "N/A"
		}
		fmt.Fprintf(&buf, "  %-*s  %s\n", maxLen, identity, label)
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

func toPolicyResultLabel(r queryPolicyResult) string {
	switch r {
	case queryPolicyResultPass:
		return "Passed"
	case queryPolicyResultFail:
		return "Failed"
	case queryPolicyResultError:
		return "Error"
	case queryPolicyResultNA:
		return "N/A"
	default:
		return "Unknown"
	}
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
