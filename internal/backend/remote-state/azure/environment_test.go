// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hcldec"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
)

func TestBackendIdentityEnvironmentVariablePrecedence(t *testing.T) {
	tests := map[string]struct {
		config map[string]interface{}
		env    map[string]string
		want   struct {
			subscriptionID string
			tenantID       string
			useOIDC        bool
			useAzureAD     bool
		}
	}{
		"configuration wins": {
			config: map[string]interface{}{
				"subscription_id":  "config-subscription",
				"tenant_id":        "config-tenant",
				"use_oidc":         false,
				"use_azuread_auth": false,
			},
			env: map[string]string{
				envARMSubscriptionIDBackend: "backend-subscription",
				"ARM_SUBSCRIPTION_ID":       "generic-subscription",
				envARMTenantIDBackend:       "backend-tenant",
				"ARM_TENANT_ID":             "generic-tenant",
				envARMUseOIDCBackend:        "true",
				"ARM_USE_OIDC":              "true",
				envARMUseAzureADBackend:     "true",
				"ARM_USE_AZUREAD":           "true",
			},
			want: struct {
				subscriptionID string
				tenantID       string
				useOIDC        bool
				useAzureAD     bool
			}{
				subscriptionID: "config-subscription",
				tenantID:       "config-tenant",
				useOIDC:        false,
				useAzureAD:     false,
			},
		},
		"backend environment wins": {
			env: map[string]string{
				envARMSubscriptionIDBackend: "backend-subscription",
				"ARM_SUBSCRIPTION_ID":       "generic-subscription",
				envARMTenantIDBackend:       "backend-tenant",
				"ARM_TENANT_ID":             "generic-tenant",
				envARMUseOIDCBackend:        "true",
				"ARM_USE_OIDC":              "false",
				envARMUseAzureADBackend:     "true",
				"ARM_USE_AZUREAD":           "false",
			},
			want: struct {
				subscriptionID string
				tenantID       string
				useOIDC        bool
				useAzureAD     bool
			}{
				subscriptionID: "backend-subscription",
				tenantID:       "backend-tenant",
				useOIDC:        true,
				useAzureAD:     true,
			},
		},
		"generic environment fallback": {
			env: map[string]string{
				"ARM_SUBSCRIPTION_ID": "generic-subscription",
				"ARM_TENANT_ID":       "generic-tenant",
				"ARM_USE_OIDC":        "true",
				"ARM_USE_AZUREAD":     "true",
			},
			want: struct {
				subscriptionID string
				tenantID       string
				useOIDC        bool
				useAzureAD     bool
			}{
				subscriptionID: "generic-subscription",
				tenantID:       "generic-tenant",
				useOIDC:        true,
				useAzureAD:     true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clearBackendIdentityEnvironment(t)
			setEnvironment(t, test.env)

			data := testBackendData(t, test.config)
			if got := data.String("subscription_id"); got != test.want.subscriptionID {
				t.Errorf("wrong subscription ID: got %q, want %q", got, test.want.subscriptionID)
			}
			if got := data.String("tenant_id"); got != test.want.tenantID {
				t.Errorf("wrong tenant ID: got %q, want %q", got, test.want.tenantID)
			}
			if got := data.Bool("use_oidc"); got != test.want.useOIDC {
				t.Errorf("wrong use_oidc: got %t, want %t", got, test.want.useOIDC)
			}
			if got := data.Bool("use_azuread_auth"); got != test.want.useAzureAD {
				t.Errorf("wrong use_azuread_auth: got %t, want %t", got, test.want.useAzureAD)
			}
		})
	}
}

func TestClientIDEnvironmentVariablePrecedence(t *testing.T) {
	t.Run("configuration wins", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		t.Setenv(envARMClientIDBackend, "backend-client")
		t.Setenv("ARM_CLIENT_ID", "generic-client")

		data := testBackendData(t, map[string]interface{}{"client_id": "config-client"})
		got, err := getClientId(&data)
		if err != nil {
			t.Fatal(err)
		}
		if *got != "config-client" {
			t.Fatalf("wrong client ID: got %q, want %q", *got, "config-client")
		}
	})

	t.Run("backend environment wins", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		t.Setenv(envARMClientIDBackend, "backend-client")
		t.Setenv("ARM_CLIENT_ID", "generic-client")

		data := testBackendData(t, nil)
		if got := data.String("client_id"); got != "" {
			t.Fatalf("client ID environment value entered prepared config: %q", got)
		}

		got, err := getClientId(&data)
		if err != nil {
			t.Fatal(err)
		}
		if *got != "backend-client" {
			t.Fatalf("wrong client ID: got %q, want %q", *got, "backend-client")
		}
	})

	t.Run("generic environment fallback", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		t.Setenv("ARM_CLIENT_ID", "generic-client")

		data := testBackendData(t, nil)
		got, err := getClientId(&data)
		if err != nil {
			t.Fatal(err)
		}
		if *got != "generic-client" {
			t.Fatalf("wrong client ID: got %q, want %q", *got, "generic-client")
		}
	})

	t.Run("backend file isolates generic pair", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		backendFile := writeTestFile(t, "backend-client-id", "backend-client")
		genericFile := writeTestFile(t, "generic-client-id", "generic-file-client")
		t.Setenv(envARMClientIDFilePathBackend, backendFile)
		t.Setenv("ARM_CLIENT_ID", "generic-client")
		t.Setenv("ARM_CLIENT_ID_FILE_PATH", genericFile)

		data := testBackendData(t, nil)
		got, err := getClientId(&data)
		if err != nil {
			t.Fatal(err)
		}
		if *got != "backend-client" {
			t.Fatalf("wrong client ID: got %q, want %q", *got, "backend-client")
		}
	})

	t.Run("generic mismatch remains an error", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		genericFile := writeTestFile(t, "generic-client-id", "file-client")
		t.Setenv("ARM_CLIENT_ID", "direct-client")
		t.Setenv("ARM_CLIENT_ID_FILE_PATH", genericFile)

		data := testBackendData(t, nil)
		_, err := getClientId(&data)
		if err == nil || !strings.Contains(err.Error(), "mismatch between supplied Client ID") {
			t.Fatalf("wrong error: %v", err)
		}
	})
}

func TestOIDCTokenEnvironmentVariablePrecedence(t *testing.T) {
	t.Run("configuration wins", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		t.Setenv(envARMOIDCTokenBackend, "backend-token")
		t.Setenv("ARM_OIDC_TOKEN", "generic-token")

		data := testBackendData(t, map[string]interface{}{"oidc_token": "config-token"})
		got, err := getOidcToken(&data)
		if err != nil {
			t.Fatal(err)
		}
		if *got != "config-token" {
			t.Fatalf("wrong OIDC token: got %q, want %q", *got, "config-token")
		}
	})

	t.Run("backend environment wins and remains runtime only", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		t.Setenv(envARMOIDCTokenBackend, "backend-token")
		t.Setenv("ARM_OIDC_TOKEN", "generic-token")

		data := testBackendData(t, nil)
		if got := data.String("oidc_token"); got != "" {
			t.Fatalf("OIDC token environment value entered prepared config: %q", got)
		}

		got, err := getOidcToken(&data)
		if err != nil {
			t.Fatal(err)
		}
		if *got != "backend-token" {
			t.Fatalf("wrong OIDC token: got %q, want %q", *got, "backend-token")
		}
	})

	t.Run("generic environment fallback", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		t.Setenv("ARM_OIDC_TOKEN", "generic-token")

		data := testBackendData(t, nil)
		got, err := getOidcToken(&data)
		if err != nil {
			t.Fatal(err)
		}
		if *got != "generic-token" {
			t.Fatalf("wrong OIDC token: got %q, want %q", *got, "generic-token")
		}
	})

	t.Run("backend file isolates generic pair", func(t *testing.T) {
		clearBackendIdentityEnvironment(t)
		backendFile := writeTestFile(t, "backend-token", "backend-token")
		genericFile := writeTestFile(t, "generic-token", "generic-file-token")
		t.Setenv(envARMOIDCTokenFilePathBackend, backendFile)
		t.Setenv("ARM_OIDC_TOKEN", "generic-token")
		t.Setenv("ARM_OIDC_TOKEN_FILE_PATH", genericFile)

		data := testBackendData(t, nil)
		got, err := getOidcToken(&data)
		if err != nil {
			t.Fatal(err)
		}
		if *got != "backend-token" {
			t.Fatalf("wrong OIDC token: got %q, want %q", *got, "backend-token")
		}
	})
}

func TestOIDCRequestEnvironmentVariablePrecedence(t *testing.T) {
	tests := map[string]struct {
		config    map[string]interface{}
		env       map[string]string
		wantURL   string
		wantToken string
	}{
		"configuration wins": {
			config: map[string]interface{}{
				"oidc_request_url":   "config-url",
				"oidc_request_token": "config-token",
			},
			env: map[string]string{
				envARMOIDCRequestURLBackend:   "backend-url",
				envARMOIDCRequestTokenBackend: "backend-token",
				"ARM_OIDC_REQUEST_URL":        "generic-url",
				"ARM_OIDC_REQUEST_TOKEN":      "generic-token",
			},
			wantURL:   "config-url",
			wantToken: "config-token",
		},
		"backend environment wins": {
			env: map[string]string{
				envARMOIDCRequestURLBackend:      "backend-url",
				envARMOIDCRequestTokenBackend:    "backend-token",
				"ARM_OIDC_REQUEST_URL":           "generic-url",
				"ARM_OIDC_REQUEST_TOKEN":         "generic-token",
				"ACTIONS_ID_TOKEN_REQUEST_URL":   "github-url",
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "github-token",
			},
			wantURL:   "backend-url",
			wantToken: "backend-token",
		},
		"generic environment fallback": {
			env: map[string]string{
				"ARM_OIDC_REQUEST_URL":           "generic-url",
				"ARM_OIDC_REQUEST_TOKEN":         "generic-token",
				"ACTIONS_ID_TOKEN_REQUEST_URL":   "github-url",
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "github-token",
			},
			wantURL:   "generic-url",
			wantToken: "generic-token",
		},
		"github actions platform fallback": {
			env: map[string]string{
				"ACTIONS_ID_TOKEN_REQUEST_URL":   "github-url",
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "github-token",
				"SYSTEM_OIDCREQUESTURI":          "ado-url",
				"SYSTEM_ACCESSTOKEN":             "ado-token",
			},
			wantURL:   "github-url",
			wantToken: "github-token",
		},
		"azure pipelines platform fallback": {
			env: map[string]string{
				"SYSTEM_OIDCREQUESTURI": "ado-url",
				"SYSTEM_ACCESSTOKEN":    "ado-token",
			},
			wantURL:   "ado-url",
			wantToken: "ado-token",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clearBackendIdentityEnvironment(t)
			setEnvironment(t, test.env)

			data := testBackendData(t, test.config)
			if test.config == nil {
				if got := data.String("oidc_request_url"); got != "" {
					t.Fatalf("OIDC request URL environment value entered prepared config: %q", got)
				}
				if got := data.String("oidc_request_token"); got != "" {
					t.Fatalf("OIDC request token environment value entered prepared config: %q", got)
				}
			}
			if got := getOidcRequestURL(&data); got != test.wantURL {
				t.Errorf("wrong OIDC request URL: got %q, want %q", got, test.wantURL)
			}
			if got := getOidcRequestToken(&data); got != test.wantToken {
				t.Errorf("wrong OIDC request token: got %q, want %q", got, test.wantToken)
			}
		})
	}
}

func TestADOPipelineServiceConnectionEnvironmentVariablePrecedence(t *testing.T) {
	tests := map[string]struct {
		config map[string]interface{}
		env    map[string]string
		want   string
	}{
		"configuration wins": {
			config: map[string]interface{}{
				"ado_pipeline_service_connection_id": "config-service-connection",
			},
			env: map[string]string{
				envARMADOPipelineServiceConnectionIDBackend: "backend-service-connection",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID":    "generic-service-connection",
			},
			want: "config-service-connection",
		},
		"canonical backend environment wins": {
			env: map[string]string{
				envARMADOPipelineServiceConnectionIDBackend: "backend-service-connection",
				envARMOIDCAzureServiceConnectionIDBackend:   "legacy-backend-service-connection",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID":    "generic-service-connection",
			},
			want: "backend-service-connection",
		},
		"legacy backend environment fallback": {
			env: map[string]string{
				envARMOIDCAzureServiceConnectionIDBackend: "legacy-backend-service-connection",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID":  "generic-service-connection",
			},
			want: "legacy-backend-service-connection",
		},
		"canonical generic environment fallback": {
			env: map[string]string{
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID":  "generic-service-connection",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID":    "legacy-generic-service-connection",
				"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID": "task-service-connection",
			},
			want: "generic-service-connection",
		},
		"legacy generic environment fallback": {
			env: map[string]string{
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID":    "legacy-generic-service-connection",
				"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID": "task-service-connection",
			},
			want: "legacy-generic-service-connection",
		},
		"azure task environment fallback": {
			env: map[string]string{
				"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID": "task-service-connection",
			},
			want: "task-service-connection",
		},
		"backend identity does not inherit provider service connection": {
			env: map[string]string{
				envARMClientIDBackend:                     "backend-client",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID":  "provider-service-connection",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID":    "legacy-provider-service-connection",
				"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID": "task-service-connection",
			},
			want: "",
		},
		"backend direct token does not inherit provider service connection": {
			env: map[string]string{
				envARMOIDCTokenBackend:                   "backend-token",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID": "provider-service-connection",
			},
			want: "",
		},
		"backend service connection remains available for separate identity": {
			env: map[string]string{
				envARMClientIDBackend:                       "backend-client",
				envARMADOPipelineServiceConnectionIDBackend: "backend-service-connection",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID":    "provider-service-connection",
			},
			want: "backend-service-connection",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clearBackendIdentityEnvironment(t)
			setEnvironment(t, test.env)

			data := testBackendData(t, test.config)
			if got := getADOPipelineServiceConnectionID(&data); got != test.want {
				t.Fatalf("wrong service connection ID: got %q, want %q", got, test.want)
			}
		})
	}
}

func testBackendData(t *testing.T, config map[string]interface{}) backendbase.SDKLikeData {
	t.Helper()

	raw := map[string]interface{}{
		"storage_account_name": "testaccount",
		"container_name":       "testcontainer",
		"key":                  "test.tfstate",
	}
	for name, value := range config {
		raw[name] = value
	}

	b := New()
	body := backend.TestWrapConfig(raw)
	configVal, diags := hcldec.Decode(body, b.ConfigSchema().DecoderSpec(), nil)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	prepared, prepareDiags := b.PrepareConfig(configVal)
	if prepareDiags.HasErrors() {
		t.Fatal(prepareDiags.ErrWithWarnings())
	}
	return backendbase.NewSDKLikeData(prepared)
}

func clearBackendIdentityEnvironment(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		envARMSubscriptionIDBackend,
		"ARM_SUBSCRIPTION_ID",
		envARMTenantIDBackend,
		"ARM_TENANT_ID",
		envARMClientIDBackend,
		"ARM_CLIENT_ID",
		envARMClientIDFilePathBackend,
		"ARM_CLIENT_ID_FILE_PATH",
		envARMUseOIDCBackend,
		"ARM_USE_OIDC",
		envARMUseAzureADBackend,
		"ARM_USE_AZUREAD",
		envARMOIDCRequestURLBackend,
		"ARM_OIDC_REQUEST_URL",
		"ACTIONS_ID_TOKEN_REQUEST_URL",
		"SYSTEM_OIDCREQUESTURI",
		envARMOIDCRequestTokenBackend,
		"ARM_OIDC_REQUEST_TOKEN",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN",
		"SYSTEM_ACCESSTOKEN",
		envARMOIDCTokenBackend,
		"ARM_OIDC_TOKEN",
		envARMOIDCTokenFilePathBackend,
		"ARM_OIDC_TOKEN_FILE_PATH",
		envARMADOPipelineServiceConnectionIDBackend,
		envARMOIDCAzureServiceConnectionIDBackend,
		"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID",
		"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID",
		"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID",
	} {
		t.Setenv(name, "")
	}
}

func setEnvironment(t *testing.T, values map[string]string) {
	t.Helper()
	for name, value := range values {
		t.Setenv(name, value)
	}
}

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
