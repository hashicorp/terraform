// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (b *Backend) PrepareConfig(configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
	// Preserve the raw values while validating the schema, before choosing environment sources.
	base := b.Base
	base.SDKLikeDefaults = nil
	configVal, diags := base.PrepareConfig(configVal)
	if diags.HasErrors() {
		return configVal, diags
	}
	data := backendbase.NewSDKLikeData(configVal)
	defaults := maps.Clone(b.SDKLikeDefaults)

	// A direct value and its file alternative must come from the same source.
	for _, pair := range [][2]string{
		{"client_id", "client_id_file_path"},
		{"oidc_token", "oidc_token_file_path"},
	} {
		if data.String(pair[0]) != "" || data.String(pair[1]) != "" {
			for _, attr := range pair {
				def := defaults[attr]
				def.EnvVars = nil
				defaults[attr] = def
			}
		} else if hasBackendEnvironmentValue(defaults, pair[:]...) {
			useBackendEnvironmentOnly(defaults, pair[:]...)
		}
	}

	// Selected backend-specific OIDC credentials must not inherit the provider's assertion or service connection.
	for _, attr := range []string{"oidc_token", "oidc_token_file_path", "oidc_request_url", "oidc_request_token"} {
		if data.String(attr) == "" && hasBackendEnvironmentValue(defaults, attr) {
			useBackendEnvironmentOnly(defaults, "oidc_token", "oidc_token_file_path", "ado_pipeline_service_connection_id")
			break
		}
	}

	serviceConnectionID := backendbase.SDKLikeEnvDefault(
		data.String("ado_pipeline_service_connection_id"),
		defaults["ado_pipeline_service_connection_id"].EnvVars...,
	)
	if serviceConnectionID == "" {
		// The SDK's Azure Pipelines request flow requires a service connection;
		// its GitHub request flow cannot consume SYSTEM_* endpoints or responses.
		for _, attr := range []string{"oidc_request_url", "oidc_request_token"} {
			def := defaults[attr]
			def.EnvVars = slices.DeleteFunc(slices.Clone(def.EnvVars), func(name string) bool {
				return strings.HasPrefix(name, "SYSTEM_")
			})
			defaults[attr] = def
		}
	}

	prepared, err := defaults.ApplyTo(configVal)
	if err != nil {
		diags = diags.Append(err)
	}
	return prepared, diags
}

func hasBackendEnvironmentValue(defaults backendbase.SDKLikeDefaults, attrs ...string) bool {
	for _, attr := range attrs {
		for _, name := range defaults[attr].EnvVars {
			if strings.HasSuffix(name, "_BACKEND") && os.Getenv(name) != "" {
				return true
			}
		}
	}
	return false
}

func useBackendEnvironmentOnly(defaults backendbase.SDKLikeDefaults, attrs ...string) {
	for _, attr := range attrs {
		def := defaults[attr]
		def.EnvVars = slices.DeleteFunc(slices.Clone(def.EnvVars), func(name string) bool {
			return !strings.HasSuffix(name, "_BACKEND")
		})
		defaults[attr] = def
	}
}
