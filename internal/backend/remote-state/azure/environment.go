// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"maps"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (b *Backend) PrepareConfig(configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
	// Resolve strict mode before reading other environment defaults.
	base := b.Base
	base.SDKLikeDefaults = backendbase.SDKLikeDefaults{
		"backend_environment_variable_strict_mode": b.SDKLikeDefaults["backend_environment_variable_strict_mode"],
	}
	configVal, diags := base.PrepareConfig(configVal)
	if diags.HasErrors() {
		return configVal, diags
	}

	strictMode := configVal.GetAttr("backend_environment_variable_strict_mode").True()
	defaults := maps.Clone(b.SDKLikeDefaults)
	if strictMode {
		for attr, def := range defaults {
			var envNames []string
			for _, name := range def.EnvVars {
				if strings.HasPrefix(name, "ARM_BACKEND_") ||
					name == "ACTIONS_ID_TOKEN_REQUEST_URL" ||
					name == "ACTIONS_ID_TOKEN_REQUEST_TOKEN" ||
					name == "SYSTEM_OIDCREQUESTURI" ||
					name == "SYSTEM_ACCESSTOKEN" {
					envNames = append(envNames, name)
				}
			}
			def.EnvVars = envNames
			defaults[attr] = def
		}
		// Missing backend credentials must not fall back to an existing Azure CLI login.
		cliDefault := defaults["use_cli"]
		cliDefault.Fallback = "false"
		defaults["use_cli"] = cliDefault
	}

	prepared, err := defaults.ApplyTo(configVal)
	if err != nil {
		diags = diags.Append(err)
	}
	return prepared, diags
}
