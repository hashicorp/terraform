// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package httpclient

import (
	"net/http"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

// New returns the DefaultPooledClient from the cleanhttp
// package that will also send a Terraform User-Agent string.
func New() *http.Client {
	cli := cleanhttp.DefaultPooledClient()
	cli.Transport = &userAgentRoundTripper{
		userAgent: UserAgentString(),
		inner:     cli.Transport,
	}
	return cli
}
