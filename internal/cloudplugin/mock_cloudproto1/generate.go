// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:generate go tool go.uber.org/mock/mockgen -destination mock.go github.com/hashicorp/terraform/internal/cloudplugin/cloudproto1 CommandServiceClient,CommandService_ExecuteClient

package mock_cloudproto1
