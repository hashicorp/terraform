// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"math/rand"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
)

func ephemeralRandomNumberSchema() providers.Schema {
	return providers.Schema{
		Block: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"value": {Type: cty.String, Computed: true},
			},
		},
	}
}

func openEphemeralRandomNumber(req providers.OpenEphemeralRequest) providers.OpenEphemeralResponse {
	result := rand.NormFloat64()
	return providers.OpenEphemeralResponse{
		Result: cty.ObjectVal(map[string]cty.Value{
			"value": cty.NumberFloatVal(result),
		}),
	}
}

func renewEphemeralRandomNumber(req providers.RenewEphemeralRequest) providers.RenewEphemeralResponse {
	// This resource type does not need renewing, but if we get asked to do
	// it for some reason then we'll just say it succeeded.
	return providers.RenewEphemeralResponse{}
}

func closeEphemeralRandomNumber(req providers.CloseEphemeralRequest) providers.CloseEphemeralResponse {
	// This resource type does not need closing because it isn't really
	// backed by any long-lived object, so we'll just say that closing it
	// succeeded even though we aren't really doing anything.
	return providers.CloseEphemeralResponse{}
}
