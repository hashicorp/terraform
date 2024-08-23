// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stubs

import (
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// offlineProvider is a stub provider that is used in place of a provider that
// is not configured  and should never be configured by the current Terraform
// configuration.
//
// The only functionality that should be called on an offlineProvider are
// provider function calls and move resource state.
//
// For everything else, Stacks should have provided a pre-configured provider
// that should be used instead.
type offlineProvider struct {
	unconfiguredClient providers.Interface
}

func OfflineProvider(unconfiguredClient providers.Interface) providers.Interface {
	return &offlineProvider{
		unconfiguredClient: unconfiguredClient,
	}
}

func (o *offlineProvider) GetProviderSchema() providers.GetProviderSchemaResponse {
	// We do actually use the schema to work out which functions are available
	// and whether cross-resource moves are even supported.
	return o.unconfiguredClient.GetProviderSchema()
}

func (o *offlineProvider) ValidateProviderConfig(request providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called ValidateProviderConfig on an unconfigured provider",
		"Cannot validate provider configuration because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ValidateProviderConfigResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) ValidateResourceConfig(request providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called ValidateResourceConfig on an unconfigured provider",
		"Cannot validate resource configuration because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ValidateResourceConfigResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) ValidateDataResourceConfig(request providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called ValidateDataResourceConfig on an unconfigured provider",
		"Cannot validate data source configuration because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ValidateDataResourceConfigResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) UpgradeResourceState(request providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called UpgradeResourceState on an unconfigured provider",
		"Cannot upgrade the state of this resource because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.UpgradeResourceStateResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) ConfigureProvider(request providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called ConfigureProvider on an unconfigured provider",
		"Cannot configure this provider because it is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ConfigureProviderResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) Stop() error {
	// pass the stop call to the underlying unconfigured client
	return o.unconfiguredClient.Stop()
}

func (o *offlineProvider) ReadResource(request providers.ReadResourceRequest) providers.ReadResourceResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called ReadResource on an unconfigured provider",
		"Cannot read from this resource because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ReadResourceResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) PlanResourceChange(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called PlanResourceChange on an unconfigured provider",
		"Cannot plan changes to this resource because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.PlanResourceChangeResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) ApplyResourceChange(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called ApplyResourceChange on an unconfigured provider",
		"Cannot apply changes to this resource because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ApplyResourceChangeResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) ImportResourceState(request providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called ImportResourceState on an unconfigured provider",
		"Cannot import an existing object into this resource because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ImportResourceStateResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) MoveResourceState(request providers.MoveResourceStateRequest) providers.MoveResourceStateResponse {
	return o.unconfiguredClient.MoveResourceState(request)
}

func (o *offlineProvider) ReadDataSource(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Called ReadDataSource on an unconfigured provider",
		"Cannot read from this data source because this provider is not configured. This is a bug in Terraform - please report it.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ReadDataSourceResponse{
		Diagnostics: diags,
	}
}

func (o *offlineProvider) CallFunction(request providers.CallFunctionRequest) providers.CallFunctionResponse {
	return o.unconfiguredClient.CallFunction(request)
}

func (o *offlineProvider) Close() error {
	// pass the close call to the underlying unconfigured client
	return o.unconfiguredClient.Close()
}
