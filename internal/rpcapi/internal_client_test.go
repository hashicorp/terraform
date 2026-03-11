// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi_test

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/terraform/internal/rpcapi"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/setup"
)

func TestInternalClientOpenClose(t *testing.T) {
	ctx := context.Background()
	client, err := rpcapi.NewInternalClient(ctx, &setup.ClientCapabilities{})
	if err != nil {
		t.Error(err)
	}

	t.Logf("server capabilities: %s", spew.Sdump(client.ServerCapabilities()))

	err = client.Close(ctx)
	if err != nil {
		t.Error(err)
	}
}
