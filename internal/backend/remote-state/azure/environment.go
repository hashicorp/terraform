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
				if strings.HasPrefix(name, "ARM_BACKEND_") {
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

		data := backendbase.NewSDKLikeData(configVal)
		serviceConnectionID := backendbase.SDKLikeEnvDefault(
			data.String("ado_pipeline_service_connection_id"),
			defaults["ado_pipeline_service_connection_id"].EnvVars...,
		)
		if strings.TrimSpace(serviceConnectionID) != "" {
			// For a backend ADO connection, use the job's OIDC URL and access token
			// after any backend request overrides.
			for _, attr := range []string{"oidc_request_url", "oidc_request_token"} {
				def := defaults[attr]
				for _, name := range b.SDKLikeDefaults[attr].EnvVars {
					if strings.HasPrefix(name, "SYSTEM_") {
						def.EnvVars = append(def.EnvVars, name)
					}
				}
				defaults[attr] = def
			}
		}
	}

	prepared, err := defaults.ApplyTo(configVal)
	if err != nil {
		diags = diags.Append(err)
	}
	return prepared, diags
}
