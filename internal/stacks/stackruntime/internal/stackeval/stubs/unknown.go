// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stubs

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var _ providers.Interface = (*unknownProvider)(nil)

// unknownProvider is a stub provider that represents a provider that is
// unknown to the current Terraform configuration. This is used when a reference
// to a provider is unknown, or the provider itself has unknown instances.
//
// An unknownProvider is only returned in the context of a provider that should
// have been configured by Stacks. This provider should not be configured again,
// or used for any dedicated offline functionality (such as moving resources and
// provider functions).
type unknownProvider struct {
	unconfiguredClient providers.Interface
}

func UnknownProvider(unconfiguredClient providers.Interface) providers.Interface {
	return &unknownProvider{
		unconfiguredClient: unconfiguredClient,
	}
}

func (u *unknownProvider) GetProviderSchema() providers.GetProviderSchemaResponse {
	// This is offline functionality, so we can hand it off to the unconfigured
	// client.
	return u.unconfiguredClient.GetProviderSchema()
}

func (u *unknownProvider) ValidateProviderConfig(request providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	// This is offline functionality, so we can hand it off to the unconfigured
	// client.
	return u.unconfiguredClient.ValidateProviderConfig(request)
}

func (u *unknownProvider) ValidateResourceConfig(request providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	// This is offline functionality, so we can hand it off to the unconfigured
	// client.
	return u.unconfiguredClient.ValidateResourceConfig(request)
}

func (u *unknownProvider) ValidateDataResourceConfig(request providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	// This is offline functionality, so we can hand it off to the unconfigured
	// client.
	return u.unconfiguredClient.ValidateDataResourceConfig(request)
}

func (u *unknownProvider) UpgradeResourceState(request providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	// This is offline functionality, so we can hand it off to the unconfigured
	// client.
	return u.unconfiguredClient.UpgradeResourceState(request)
}

func (u *unknownProvider) ConfigureProvider(request providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	// This shouldn't be called, we don't configure an unknown provider within
	// stacks and Terraform Core shouldn't call this method.
	panic("attempted to configure an unknown provider")
}

func (u *unknownProvider) Stop() error {
	// the underlying unconfiguredClient is managed elsewhere.
	return nil
}

func (u *unknownProvider) ReadResource(request providers.ReadResourceRequest) providers.ReadResourceResponse {
	if request.ClientCapabilities.DeferralAllowed {
		// For ReadResource, we'll just return the existing state and defer
		// the operation.
		return providers.ReadResourceResponse{
			NewState: request.PriorState,
			Deferred: &providers.Deferred{
				Reason: providers.DeferredReasonProviderConfigUnknown,
			},
		}
	}
	return providers.ReadResourceResponse{
		Diagnostics: []tfdiags.Diagnostic{
			tfdiags.AttributeValue(
				tfdiags.Error,
				"Provider configuration is unknown",
				"Cannot read from this data source because its associated provider configuration is unknown.",
				nil, // nil attribute path means the overall configuration block
			),
		},
	}
}

func (u *unknownProvider) PlanResourceChange(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	if request.ClientCapabilities.DeferralAllowed {
		// For PlanResourceChange, we'll kind of abuse the mocking library to
		// populate the computed values with unknown values so that future
		// operations can still be used.
		//
		// PlanComputedValuesForResource populates the computed values with
		// unknown values. This isn't the original use case for the mocking
		// library, but it is doing exactly what we need it to do.

		schema := u.GetProviderSchema().ResourceTypes[request.TypeName]
		val, diags := mocking.PlanComputedValuesForResource(request.ProposedNewState, schema.Block)
		if diags.HasErrors() {
			// All the potential errors we get back from this function are
			// related to the user badly defining mocks. We should never hit
			// this as we are just using the default behaviour.
			panic(diags.Err())
		}

		return providers.PlanResourceChangeResponse{
			PlannedState: val,
			Deferred: &providers.Deferred{
				Reason: providers.DeferredReasonProviderConfigUnknown,
			},
		}
	}
	return providers.PlanResourceChangeResponse{
		Diagnostics: []tfdiags.Diagnostic{
			tfdiags.AttributeValue(
				tfdiags.Error,
				"Provider configuration is unknown",
				"Cannot plan changes for this resource because its associated provider configuration is unknown.",
				nil, // nil attribute path means the overall configuration block
			),
		},
	}
}

func (u *unknownProvider) ApplyResourceChange(_ providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	return providers.ApplyResourceChangeResponse{
		Diagnostics: []tfdiags.Diagnostic{
			tfdiags.AttributeValue(
				tfdiags.Error,
				"Provider configuration is unknown",
				"Cannot apply changes for this resource because its associated provider configuration is unknown.",
				nil, // nil attribute path means the overall configuration block
			),
		},
	}
}

func (u *unknownProvider) ImportResourceState(request providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	if request.ClientCapabilities.DeferralAllowed {
		// For ImportResourceState, we don't have any config to work with and
		// we don't know enough to work out which value the ID corresponds to.
		//
		// We'll just return an unknown value that corresponds to the correct
		// type. Terraform should know how to handle this when it arrives
		// alongside the deferred metadata.

		schema := u.GetProviderSchema().ResourceTypes[request.TypeName]
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: request.TypeName,
					State:    cty.UnknownVal(schema.Block.ImpliedType()),
				},
			},
			Deferred: &providers.Deferred{
				Reason: providers.DeferredReasonProviderConfigUnknown,
			},
		}
	}
	return providers.ImportResourceStateResponse{
		Diagnostics: []tfdiags.Diagnostic{
			tfdiags.AttributeValue(
				tfdiags.Error,
				"Provider configuration is unknown",
				"Cannot import an existing object into this resource because its associated provider configuration is unknown.",
				nil, // nil attribute path means the overall configuration block
			),
		},
	}
}

func (u *unknownProvider) MoveResourceState(request providers.MoveResourceStateRequest) providers.MoveResourceStateResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called MoveResourceState on an unknown provider",
		"Terraform called MoveResourceState on an unknown provider. This is a bug in Terraform - please report this error.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.MoveResourceStateResponse{
		Diagnostics: diags,
	}
}

func (u *unknownProvider) ReadDataSource(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	if request.ClientCapabilities.DeferralAllowed {
		// For ReadDataSource, we'll kind of abuse the mocking library to
		// populate the computed values with unknown values so that future
		// operations can still be used.
		//
		// PlanComputedValuesForResource populates the computed values with
		// unknown values. This isn't the original use case for the mocking
		// library, but it is doing exactly what we need it to do.

		schema := u.GetProviderSchema().ResourceTypes[request.TypeName]
		val, diags := mocking.PlanComputedValuesForResource(request.Config, schema.Block)
		if diags.HasErrors() {
			// All the potential errors we get back from this function are
			// related to the user badly defining mocks. We should never hit
			// this as we are just using the default behaviour.
			panic(diags.Err())
		}

		return providers.ReadDataSourceResponse{
			State: val,
			Deferred: &providers.Deferred{
				Reason: providers.DeferredReasonProviderConfigUnknown,
			},
		}
	}
	return providers.ReadDataSourceResponse{
		Diagnostics: []tfdiags.Diagnostic{
			tfdiags.AttributeValue(
				tfdiags.Error,
				"Provider configuration is unknown",
				"Cannot read from this data source because its associated provider configuration is unknown.",
				nil, // nil attribute path means the overall configuration block
			),
		},
	}
}

func (u *unknownProvider) CallFunction(request providers.CallFunctionRequest) providers.CallFunctionResponse {
	return providers.CallFunctionResponse{
		Err: fmt.Errorf("CallFunction shouldn't be called on an unknown provider; this is a bug in Terraform - please report this error"),
	}
}

func (u *unknownProvider) Close() error {
	// the underlying unconfiguredClient is managed elsewhere.
	return nil
}
