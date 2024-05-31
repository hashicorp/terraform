// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stubs

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var _ providers.Interface = (*unknownProvider)(nil)

// unknownProvider is a stub provider that represents a provider that is
// unknown to the current Terraform configuration. This is used when a reference
// to a provider is unknown, or the provider itself has unknown instances.
//
// This provider wraps an unconfigured provider client, which is used to handle
// offline functionality.
//
// TODO: We can return more specific values than cty.DynamicVal.
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
		return providers.ReadResourceResponse{
			NewState: cty.DynamicVal,
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
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.DynamicVal,
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

func (u *unknownProvider) ApplyResourceChange(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
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
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: request.TypeName,
					State:    cty.DynamicVal,
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
	// This is offline functionality, so we can hand it off to the unconfigured
	// client.
	return u.unconfiguredClient.MoveResourceState(request)
}

func (u *unknownProvider) ReadDataSource(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	if request.ClientCapabilities.DeferralAllowed {
		return providers.ReadDataSourceResponse{
			State: cty.DynamicVal,
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
	// This is offline functionality, so we can hand it off to the unconfigured
	// client.
	return u.unconfiguredClient.CallFunction(request)
}

func (u *unknownProvider) Close() error {
	// the underlying unconfiguredClient is managed elsewhere.
	return nil
}
