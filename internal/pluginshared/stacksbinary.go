// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pluginshared

import (
	"context"
	"fmt"
	"net/url"

	svchost "github.com/hashicorp/terraform-svchost"
)

// StacksBinaryManager downloads, caches, and returns information about the
// terraform-stacksplugin binary downloaded from the specified backend.
type StacksBinaryManager struct {
	BinaryManager
}

// NewStacksBinaryManager initializes a new BinaryManager to broker data between the
// specified directory location containing stacksplugin package data and a
// HCP Terraform backend URL.
func NewStacksBinaryManager(ctx context.Context, stacksPluginDataDir, overridePath string, serviceURL *url.URL, goos, arch string) (*StacksBinaryManager, error) {
	client, err := NewStacksPluginClient(ctx, serviceURL)
	if err != nil {
		return nil, fmt.Errorf("could not initialize stacksplugin version manager: %w", err)
	}

	return &StacksBinaryManager{
		BinaryManager{
			pluginDataDir: stacksPluginDataDir,
			overridePath:  overridePath,
			host:          svchost.Hostname(serviceURL.Host),
			client:        client,
			binaryName:    "terraform-stacksplugin",
			pluginName:    "stacksplugin",
			goos:          goos,
			arch:          arch,
			ctx:           ctx,
		}}, nil
}
