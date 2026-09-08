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
		"strict_mode": b.SDKLikeDefaults["strict_mode"],
	}
	configVal, diags := base.PrepareConfig(configVal)
	if diags.HasErrors() {
		return configVal, diags
	}

	strictMode := configVal.GetAttr("strict_mode").True()
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
		for _, pair := range [][2]string{
			{"client_id", "client_id_file_path"},
			{"client_secret", "client_secret_file_path"},
			{"client_certificate", "client_certificate_path"},
			{"oidc_token", "oidc_token_file_path"},
		} {
			if anyStringSet(data, pair[:]...) {
				disableEnvironmentDefaults(defaults, pair[:]...)
			}
		}

		// Explicit OIDC inputs select the method; conflicting explicit inputs remain for validation.
		if anyStringSet(data, "oidc_token", "oidc_token_file_path") || data.Bool("use_aks_workload_identity") {
			disableEnvironmentDefaults(defaults, "oidc_request_url", "oidc_request_token", "ado_pipeline_service_connection_id")
		}
		if anyStringSet(data, "oidc_request_url", "oidc_request_token", "ado_pipeline_service_connection_id") {
			disableEnvironmentDefaults(defaults, "oidc_token", "oidc_token_file_path", "use_aks_workload_identity")
		}

		serviceConnectionID := backendbase.SDKLikeEnvDefault(
			data.String("ado_pipeline_service_connection_id"),
			defaults["ado_pipeline_service_connection_id"].EnvVars...,
		)
		if strings.TrimSpace(serviceConnectionID) != "" {
			// A selected ADO connection can use the current job's native broker, after ARM request overrides.
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
		return prepared, diags
	}
	if strictMode {
		data := backendbase.NewSDKLikeData(prepared)
		hasAssertion := anyStringSet(data, "oidc_token", "oidc_token_file_path") || data.Bool("use_aks_workload_identity")
		hasRequest := anyStringSet(data, "oidc_request_url", "oidc_request_token", "ado_pipeline_service_connection_id")
		if hasAssertion && hasRequest {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Conflicting OIDC authentication settings",
				"When strict_mode is enabled, choose either an OIDC assertion (oidc_token, oidc_token_file_path, or use_aks_workload_identity) or an OIDC request (oidc_request_url, oidc_request_token, or ado_pipeline_service_connection_id), not both.",
			))
		}
	}
	return prepared, diags
}

func anyStringSet(data backendbase.SDKLikeData, attrs ...string) bool {
	for _, attr := range attrs {
		if data.String(attr) != "" {
			return true
		}
	}
	return false
}

func disableEnvironmentDefaults(defaults backendbase.SDKLikeDefaults, attrs ...string) {
	for _, attr := range attrs {
		def := defaults[attr]
		def.EnvVars = nil
		defaults[attr] = def
	}
}
