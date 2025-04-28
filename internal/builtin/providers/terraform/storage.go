// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/providers"
)

func (p *Provider) ValidateStorageConfig(req providers.ValidateStorageConfigRequest) providers.ValidateStorageConfigResponse {
	var resp providers.ValidateStorageConfigResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported storage type %q", req.TypeName))
	return resp
}

func (p *Provider) ConfigureStorage(req providers.ConfigureStorageRequest) providers.ConfigureStorageResponse {
	var resp providers.ConfigureStorageResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported storage type %q", req.TypeName))
	return resp
}

func (p *Provider) LockState(req providers.LockStateRequest) providers.LockStateResponse {
	var resp providers.LockStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported storage type %q", req.TypeName))
	return resp
}

func (p *Provider) UnlockState(req providers.UnlockStateRequest) providers.UnlockStateResponse {
	var resp providers.UnlockStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported storage type %q", req.TypeName))
	return resp
}

func (p *Provider) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	var resp providers.GetStatesResponse
	resp.StateIds = nil // No states when no state storage is implemented, even `default`
	return resp
}

func (p *Provider) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	var resp providers.DeleteStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported storage type %q", req.TypeName))
	return resp
}
