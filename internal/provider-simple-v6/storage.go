package simple

import (
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (s simple) ValidateStorageConfig(req providers.ValidateStorageConfigRequest) providers.ValidateStorageConfigResponse {
	// TODO
	var diags tfdiags.Diagnostics
	return providers.ValidateStorageConfigResponse{
		Diagnostics: diags,
	}
}

func (s simple) ConfigureStorage(req providers.ConfigureStorageRequest) providers.ConfigureStorageResponse {
	// TODO
	var diags tfdiags.Diagnostics
	return providers.ConfigureStorageResponse{
		Diagnostics: diags,
	}
}

func (s simple) LockState(req providers.LockStateRequest) providers.LockStateResponse {
	// TODO
	return providers.LockStateResponse{}
}

func (s simple) UnlockState(req providers.UnlockStateRequest) providers.UnlockStateResponse {
	// TODO
	return providers.UnlockStateResponse{}
}

func (s simple) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	// TODO
	return providers.GetStatesResponse{}
}

func (s simple) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	// TODO
	return providers.DeleteStateResponse{}
}
