// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "github.com/hashicorp/terraform/internal/providers"

// unimplementedProviderInterface implements all methods of provider.Interface but they panic and report they aren't implemented.
//
// This allows state store implementations to be passed around as provider.Interface instances in the builtin provider.
// To use unimplementedProviderInterface embed it into a struct implementing state storage.
type unimplementedProviderInterface struct{}

var _ providers.Interface = &InMemStoreSingle{}

func (unimplementedProviderInterface) GetProviderSchema() providers.GetProviderSchemaResponse {
	panic("GetProviderSchema isn't implemented")
}
func (unimplementedProviderInterface) GetResourceIdentitySchemas() providers.GetResourceIdentitySchemasResponse {
	panic("GetResourceIdentitySchemas isn't implemented")
}
func (unimplementedProviderInterface) ValidateProviderConfig(providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	panic("ValidateProviderConfig isn't implemented")
}
func (unimplementedProviderInterface) ValidateResourceConfig(providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	panic("ValidateResourceConfig isn't implemented")
}
func (unimplementedProviderInterface) ValidateDataResourceConfig(providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	panic("ValidateDataResourceConfig isn't implemented")
}
func (unimplementedProviderInterface) ValidateEphemeralResourceConfig(providers.ValidateEphemeralResourceConfigRequest) providers.ValidateEphemeralResourceConfigResponse {
	panic("ValidateEphemeralResourceConfig isn't implemented")
}
func (unimplementedProviderInterface) ValidateListResourceConfig(providers.ValidateListResourceConfigRequest) providers.ValidateListResourceConfigResponse {
	panic("ValidateListResourceConfig isn't implemented")
}
func (unimplementedProviderInterface) UpgradeResourceState(providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	panic("UpgradeResourceState isn't implemented")
}
func (unimplementedProviderInterface) UpgradeResourceIdentity(providers.UpgradeResourceIdentityRequest) providers.UpgradeResourceIdentityResponse {
	panic("UpgradeResourceIdentity isn't implemented")
}
func (unimplementedProviderInterface) ConfigureProvider(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	panic("ConfigureProvider isn't implemented")
}
func (unimplementedProviderInterface) Stop() error {
	panic("Stop isn't implemented")
}
func (unimplementedProviderInterface) ReadResource(providers.ReadResourceRequest) providers.ReadResourceResponse {
	panic("ReadResource isn't implemented")
}
func (unimplementedProviderInterface) PlanResourceChange(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	panic("PlanResourceChange isn't implemented")
}
func (unimplementedProviderInterface) ApplyResourceChange(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	panic("ApplyResourceChange isn't implemented")
}
func (unimplementedProviderInterface) ImportResourceState(providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	panic("ImportResourceState isn't implemented")
}
func (unimplementedProviderInterface) GenerateResourceConfig(providers.GenerateResourceConfigRequest) providers.GenerateResourceConfigResponse {
	panic("GenerateResourceConfig isn't implemented")
}
func (unimplementedProviderInterface) MoveResourceState(providers.MoveResourceStateRequest) providers.MoveResourceStateResponse {
	panic("MoveResourceState isn't implemented")
}
func (unimplementedProviderInterface) ReadDataSource(providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	panic("ReadDataSource isn't implemented")
}
func (unimplementedProviderInterface) OpenEphemeralResource(providers.OpenEphemeralResourceRequest) providers.OpenEphemeralResourceResponse {
	panic("OpenEphemeralResource isn't implemented")
}
func (unimplementedProviderInterface) RenewEphemeralResource(providers.RenewEphemeralResourceRequest) providers.RenewEphemeralResourceResponse {
	panic("RenewEphemeralResource isn't implemented")
}
func (unimplementedProviderInterface) CloseEphemeralResource(providers.CloseEphemeralResourceRequest) providers.CloseEphemeralResourceResponse {
	panic("CloseEphemeralResource isn't implemented")
}
func (unimplementedProviderInterface) CallFunction(providers.CallFunctionRequest) providers.CallFunctionResponse {
	panic("CallFunction isn't implemented")
}
func (unimplementedProviderInterface) ListResource(providers.ListResourceRequest) providers.ListResourceResponse {
	panic("ListResource isn't implemented")
}
func (unimplementedProviderInterface) ValidateStateStoreConfig(providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
	panic("ValidateStateStoreConfig isn't implemented")
}
func (unimplementedProviderInterface) ConfigureStateStore(providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
	panic("ConfigureStateStore isn't implemented")
}
func (unimplementedProviderInterface) ReadStateBytes(providers.ReadStateBytesRequest) providers.ReadStateBytesResponse {
	panic("ReadStateBytes isn't implemented")
}
func (unimplementedProviderInterface) WriteStateBytes(providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
	panic("WriteStateBytes isn't implemented")
}
func (unimplementedProviderInterface) LockState(providers.LockStateRequest) providers.LockStateResponse {
	panic("LockState isn't implemented")
}
func (unimplementedProviderInterface) UnlockState(providers.UnlockStateRequest) providers.UnlockStateResponse {
	panic("UnlockState isn't implemented")
}
func (unimplementedProviderInterface) GetStates(providers.GetStatesRequest) providers.GetStatesResponse {
	panic("GetStates isn't implemented")
}
func (unimplementedProviderInterface) DeleteState(providers.DeleteStateRequest) providers.DeleteStateResponse {
	panic("DeleteState isn't implemented")
}
func (unimplementedProviderInterface) PlanAction(providers.PlanActionRequest) providers.PlanActionResponse {
	panic("PlanAction isn't implemented")
}
func (unimplementedProviderInterface) InvokeAction(providers.InvokeActionRequest) providers.InvokeActionResponse {
	panic("InvokeAction isn't implemented")
}
func (unimplementedProviderInterface) ValidateActionConfig(providers.ValidateActionConfigRequest) providers.ValidateActionConfigResponse {
	panic("ValidateActionConfig isn't implemented")
}
func (unimplementedProviderInterface) Close() error {
	panic("Close isn't implemented")
}
