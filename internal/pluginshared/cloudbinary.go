// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pluginshared

import (
	"context"
	"fmt"
	"net/url"

	svchost "github.com/hashicorp/terraform-svchost"
)

// CloudBinaryManager downloads, caches, and returns information about the
// terraform-cloudplugin binary downloaded from the specified backend.
type CloudBinaryManager struct {
	BinaryManager
}

// NewCloudBinaryManager initializes a new BinaryManager to broker data between the
// specified directory location containing cloudplugin package data and a
// HCP Terraform backend URL.
func NewCloudBinaryManager(ctx context.Context, cloudPluginDataDir, overridePath string, serviceURL *url.URL, goos, arch string) (*CloudBinaryManager, error) {
	client, err := NewCloudPluginClient(ctx, serviceURL)
	if err != nil {
		return nil, fmt.Errorf("could not initialize cloudplugin version manager: %w", err)
	}

	return &CloudBinaryManager{
		BinaryManager{
			pluginDataDir: cloudPluginDataDir,
			overridePath:  overridePath,
			host:          svchost.Hostname(serviceURL.Host),
			client:        client,
			binaryName:    "terraform-cloudplugin",
			pluginName:    "cloudplugin",
			goos:          goos,
			arch:          arch,
			ctx:           ctx,
		}}, nil
}
