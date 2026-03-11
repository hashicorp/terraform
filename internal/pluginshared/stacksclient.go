// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pluginshared

import (
	"context"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/logging"
)

// NewStacksPluginClient creates a new client for downloading and verifying
// terraform-stacks plugin archives
func NewStacksPluginClient(ctx context.Context, serviceURL *url.URL) (*BasePluginClient, error) {
	httpClient := httpclient.New()
	httpClient.Timeout = defaultRequestTimeout

	retryableClient := retryablehttp.NewClient()
	retryableClient.HTTPClient = httpClient
	retryableClient.RetryMax = 3
	retryableClient.RequestLogHook = requestLogHook
	retryableClient.Logger = logging.HCLogger()

	client := BasePluginClient{
		ctx:        ctx,
		serviceURL: serviceURL,
		httpClient: retryableClient,
		pluginName: "stacksplugin",
	}
	return &client, nil
}
