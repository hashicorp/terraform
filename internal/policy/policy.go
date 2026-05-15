// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

// Client is an interface for interacting with a policy engine.
type Client interface {
	Setup(context.Context, SetupRequest) SetupResponse
	Evaluate(context.Context, EvaluationRequest[*proto.ResourceMetadata]) EvaluationResponse
	EvaluateProvider(context.Context, EvaluationRequest[*proto.ProviderMetadata]) EvaluationResponse
	EvaluateModule(context.Context, EvaluationRequest[*proto.ModuleMetadata]) EvaluationResponse
	Stop()
}

// CallbackService is an interface for registering a callback service with a policy engine.
type CallbackService interface {
	RegisterCallbackService(context.Context) (*callback.Server, Diagnostics)
}

type (
	SetupResponse struct {
		// serverCapabilities contains the map of a policy path to the capabilities of the server.
		serverCapabilities *proto.PolicySetupResponse_ServerCapabilities

		Diagnostics Diagnostics
	}

	ServerConfiguration struct {
		// File is the path to the policy file.
		File string

		// RequiredVersion is the required version of the policy file.
		RequiredVersion string
	}
)

func (s *SetupResponse) ServerConfigurations() []ServerConfiguration {
	if s.serverCapabilities == nil {
		return nil
	}
	ret := make([]ServerConfiguration, 0, len(s.serverCapabilities.Configurations))
	for file, capabilities := range s.serverCapabilities.Configurations {
		ret = append(ret, ServerConfiguration{
			File:            file,
			RequiredVersion: capabilities.RequiredVersion,
		})
	}
	return ret
}

type (
	SetupRequest struct {
		// SourceLocations is the list of source locations to load policies from.
		SourceLocations []string

		// CallbackService is the callback service to use for policy evaluation.
		CallbackService uint32
	}

	PolicyValue struct {
		// Raw contains the Terraform value being sent to the policy engine.
		Raw cty.Value

		// RedactedPaths contains attribute paths that should be redacted when
		// displaying values from Raw.
		RedactedPaths []cty.Path
	}

	EvaluationRequest[T any] struct {
		// Target is the object being evaluated.
		Target string

		// Attrs contains the attributes of the object being evaluated.
		Attrs PolicyValue

		// PriorAttrs contains the state of the object prior to the current operation.
		PriorAttrs PolicyValue

		// Meta is additional metadata required for evaluation.
		Meta T

		Callbacks callback.Functions
	}

	EnforcementResult struct {
		Result  EvaluateResult
		Message string
		Range   *hcl.Range
		Snippet *proto.Snippet
		Policy  *Policy

		// BlockIndex is the index of the enforce block within the policy originating policy.
		BlockIndex int32

		// LocalRange is the range of the terraform object being evaluated
		LocalRange *hcl.Range
	}

	// Policy contains information about a policy block
	Policy struct {
		Result  EvaluateResult
		Address string

		// Directory is the full path to the policy file.
		Directory string

		Filename string

		Range            *hcl.Range
		PolicySetName    string
		EnforcementLevel string
	}

	// EvaluationResponse contains response from a single Evaluate RPC request.
	EvaluationResponse struct {
		// Overall is a result of all enforcements evaluated in a single Evaluate RPC request.
		Overall EvaluateResult

		// Enforcements is a slice of each enforce result in all the policies evaluated.
		Enforcements []EnforcementResult

		// Policies are the policies which were evaluated for the targeted resource.
		Policies []*Policy

		// A combination of Policy- and Enforcement-level diagnostics.
		Diagnostics Diagnostics
	}
)

func EvaluationFromProtoResponse(overall proto.EvaluateResult, policyDetails []*proto.PolicyEvaluationDetail) EvaluationResponse {
	ret := EvaluationResponse{
		Overall:      ResultFromProto(overall),
		Enforcements: make([]EnforcementResult, 0, len(policyDetails)),
		Diagnostics:  Diagnostics{},
		Policies:     make([]*Policy, 0),
	}
	for _, protoPolicy := range policyDetails {
		rng := protoPolicy.DefRange.ToHclRange()
		policy := &Policy{
			Result:           ResultFromProto(protoPolicy.Result),
			Address:          protoPolicy.Address,
			Directory:        protoPolicy.File,
			PolicySetName:    protoPolicy.PolicySetName,
			Filename:         filepath.Base(rng.Filename),
			EnforcementLevel: protoPolicy.PolicySetEnforcement,
			Range:            rng.Ptr(),
		}

		// We go through each diagnostic and attach the originating policy to it as an extra
		policyDiags := DiagsFromProto(protoPolicy.Diagnostics, policy)
		ret.Diagnostics = append(ret.Diagnostics, policyDiags...)
		ret.Policies = append(ret.Policies, policy)

		for _, enforcement := range protoPolicy.EnforceResults {
			result := EnforcementResult{
				Result:     ResultFromProto(enforcement.Result),
				Message:    enforcement.Message,
				Range:      enforcement.Range.ToHclRange().Ptr(),
				Snippet:    enforcement.Snippet,
				Policy:     policy,
				BlockIndex: enforcement.BlockIndex,
			}
			ret.Enforcements = append(ret.Enforcements, result)

			// Attach the enforce index to any diagnostics from the enforce block
			policyDiags := DiagsFromProto(enforcement.Diagnostics, policy)
			for idx := range policyDiags {
				policyDiags[idx].extra.EnforceIndex = &enforcement.BlockIndex
			}
			ret.Diagnostics = append(ret.Diagnostics, policyDiags...)
		}
	}

	return ret
}

func (r EvaluationResponse) Empty() bool {
	// The policy engine sends an allow result when the object has no matched policy, consequently
	// impliciting allowing it. However, such object really had no policy, and may not need to be rendered.
	if r.Overall == AllowResult && len(r.Diagnostics) == 0 && len(r.Enforcements) == 0 {
		return true
	}

	return false
}

func ErrorEvalFromDiags(diags []*proto.Diagnostic) EvaluationResponse {
	return EvaluationResponse{
		Overall:     PolicyErrorResult,
		Diagnostics: DiagsFromProto(diags, nil),
	}
}
