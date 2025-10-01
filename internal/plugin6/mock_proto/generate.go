// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:generate go tool go.uber.org/mock/mockgen -destination mock.go github.com/hashicorp/terraform/internal/tfplugin6 ProviderClient,Provider_InvokeActionClient

package mock_tfplugin6
