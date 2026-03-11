// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import "github.com/hashicorp/terraform/internal/providers"

// ClientCapabilities returns the client capabilities sent to the providers
// for each request. They define what this terraform instance is capable of.
func ClientCapabilities() providers.ClientCapabilities {
	return providers.ClientCapabilities{
		DeferralAllowed:            true,
		WriteOnlyAttributesAllowed: true,
	}
}
