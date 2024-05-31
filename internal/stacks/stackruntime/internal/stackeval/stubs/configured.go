// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stubs

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ConfiguredProvider is a placeholder provider used when ConfigureProvider
// on a real provider fails, so that callers can still receieve a usable client
// that will just produce placeholder values from its operations.
//
// This is essentially the cty.DynamicVal equivalent for providers.Interface,
// allowing us to follow our usual pattern that only one return path carries
// diagnostics up to the caller and all other codepaths just do their best
// to unwind with placeholder values. It's intended only for use in situations
// that would expect an already-configured provider, so it's incorrect to call
// [ConfigureProvider] on a value of this type.
//
// Some methods of this type explicitly return errors saying that the provider
// configuration was invalid, while others just optimistically do nothing at
// all. The general rule is that anything that would for a normal provider
// be expected to perform externally-visible side effects must return an error
// to be explicit that those side effects did not occur, but we can silently
// skip anything that is a Terraform-only detail.
//
// As usual with provider calls, the returned diagnostics must be annotated
// using [tfdiags.Diagnostics.InConfigBody] with the relevant configuration body
// so that they can be attributed to the appropriate configuration element.
type ConfiguredProvider struct {
	// If unknown is true then the implementation will assume it's acting
	// as a placeholder for a provider whose configuration isn't yet
	// sufficiently known to be properly instantiated, which means that
	// plan-time operations will return totally-unknown values.
	// Otherwise any operation that is supposed to perform a side-effect
	// will fail with an error saying that the provider configuration
	// is invalid.
	Unknown bool
}

var _ providers.Interface = ConfiguredProvider{}

// ApplyResourceChange implements providers.Interface.
func (ConfiguredProvider) ApplyResourceChange(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Provider configuration is invalid",
		"Cannot apply changes because this resource's associated provider configuration is invalid.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ApplyResourceChangeResponse{
		Diagnostics: diags,
	}
}

func (ConfiguredProvider) CallFunction(providers.CallFunctionRequest) providers.CallFunctionResponse {
	panic("can't call functions on the stub provider")
}

// Close implements providers.Interface.
func (ConfiguredProvider) Close() error {
	return nil
}

// ConfigureProvider implements providers.Interface.
func (ConfiguredProvider) ConfigureProvider(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	// This provider is used only in situations where ConfigureProvider on
	// a real provider fails and the recipient was expecting a configured
	// provider, so it doesn't make sense to configure it.
	panic("can't configure the stub provider")
}

// GetProviderSchema implements providers.Interface.
func (ConfiguredProvider) GetProviderSchema() providers.GetProviderSchemaResponse {
	return providers.GetProviderSchemaResponse{}
}

// ImportResourceState implements providers.Interface.
func (p ConfiguredProvider) ImportResourceState(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	var diags tfdiags.Diagnostics
	if p.Unknown {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider configuration is deferred",
			"Cannot import an existing object into this resource because its associated provider configuration is deferred to a later operation due to unknown expansion.",
			nil, // nil attribute path means the overall configuration block
		))
	} else {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider configuration is invalid",
			"Cannot import an existing object into this resource because its associated provider configuration is invalid.",
			nil, // nil attribute path means the overall configuration block
		))
	}
	return providers.ImportResourceStateResponse{
		Diagnostics: diags,
	}
}

// MoveResourceState implements providers.Interface.
func (p ConfiguredProvider) MoveResourceState(req providers.MoveResourceStateRequest) providers.MoveResourceStateResponse {
	var diags tfdiags.Diagnostics
	if p.Unknown {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider configuration is deferred",
			"Cannot move an existing object to this resource because its associated provider configuration is deferred to a later operation due to unknown expansion.",
			nil, // nil attribute path means the overall configuration block
		))
	} else {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider configuration is invalid",
			"Cannot move an existing object to this resource because its associated provider configuration is invalid.",
			nil, // nil attribute path means the overall configuration block
		))
	}
	return providers.MoveResourceStateResponse{
		Diagnostics: diags,
	}
}

// PlanResourceChange implements providers.Interface.
func (p ConfiguredProvider) PlanResourceChange(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	if p.Unknown {
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.DynamicVal,
		}
	}
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Provider configuration is invalid",
		"Cannot plan changes for this resource because its associated provider configuration is invalid.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.PlanResourceChangeResponse{
		Diagnostics: diags,
	}
}

// ReadDataSource implements providers.Interface.
func (p ConfiguredProvider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	if p.Unknown {
		return providers.ReadDataSourceResponse{
			State: cty.DynamicVal,
		}
	}
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Provider configuration is invalid",
		"Cannot read from this data source because its associated provider configuration is invalid.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ReadDataSourceResponse{
		Diagnostics: diags,
	}
}

// ReadResource implements providers.Interface.
func (ConfiguredProvider) ReadResource(req providers.ReadResourceRequest) providers.ReadResourceResponse {
	// For this one we'll just optimistically assume that the remote object
	// hasn't changed. In many cases we'll fail calling PlanResourceChange
	// right afterwards anyway, and even if not we'll get another opportunity
	// to refresh on a future run once the provider configuration is fixed.
	return providers.ReadResourceResponse{
		NewState: req.PriorState,
		Private:  req.Private,
	}
}

// Stop implements providers.Interface.
func (ConfiguredProvider) Stop() error {
	// This stub provider never actually does any real work, so there's nothing
	// for us to stop.
	return nil
}

// UpgradeResourceState implements providers.Interface.
func (p ConfiguredProvider) UpgradeResourceState(req providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	if p.Unknown {
		return providers.UpgradeResourceStateResponse{
			UpgradedState: cty.DynamicVal,
		}
	}

	// Ideally we'd just skip this altogether and echo back what the caller
	// provided, but the request is in a different serialization format than
	// the response and so only the real provider can deal with this one.
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Provider configuration is invalid",
		"Cannot decode the prior state for this resource instance because its provider configuration is invalid.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.UpgradeResourceStateResponse{
		Diagnostics: diags,
	}
}

// ValidateDataResourceConfig implements providers.Interface.
func (ConfiguredProvider) ValidateDataResourceConfig(req providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	// We'll just optimistically assume the configuration is valid, so that
	// we can progress to planning and return an error there instead.
	return providers.ValidateDataResourceConfigResponse{
		Diagnostics: nil,
	}
}

// ValidateProviderConfig implements providers.Interface.
func (ConfiguredProvider) ValidateProviderConfig(req providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	// It doesn't make sense to call this one on stubProvider, because
	// we only use stubProvider for situations where ConfigureProvider failed
	// on a real provider and we should already have called
	// ValidateProviderConfig on that provider by then anyway.
	return providers.ValidateProviderConfigResponse{
		PreparedConfig: req.Config,
		Diagnostics:    nil,
	}
}

// ValidateResourceConfig implements providers.Interface.
func (ConfiguredProvider) ValidateResourceConfig(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	// We'll just optimistically assume the configuration is valid, so that
	// we can progress to reading and return an error there instead.
	return providers.ValidateResourceConfigResponse{
		Diagnostics: nil,
	}
}
