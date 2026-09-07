// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:generate go tool go.uber.org/mock/mockgen -destination mock.go github.com/hashicorp/terraform/internal/tfplugin6 ProviderClient,Provider_InvokeActionClient,Provider_ReadStateBytesClient,Provider_WriteStateBytesClient

package mock_tfplugin6
