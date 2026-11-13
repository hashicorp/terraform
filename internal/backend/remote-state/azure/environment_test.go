// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"context"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
)

func TestBackendEnvironmentVariableStrictMode(t *testing.T) {
	tests := []struct {
		name, control string
		config        interface{}
		want          bool
		wantError     bool
	}{
		{name: "unset"},
		{name: "control enabled", control: "true", want: true},
		{name: "control disabled", control: "false"},
		{name: "explicit true wins", control: "false", config: true, want: true},
		{name: "explicit false wins", control: "true", config: false},
		{name: "explicit false ignores invalid control", control: "invalid", config: false},
		{name: "invalid control", control: "invalid", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_STRICT_MODE", test.control)
			t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_SUFFIX", "_IGNORED")
			t.Setenv("ARM_CLIENT_ID", "legacy-client")
			t.Setenv("ARM_CLIENT_ID_BACKEND", "ignored-old-name")
			b := New()
			if _, exists := b.ConfigSchema().Attributes["environment_variable_suffix"]; exists {
				t.Fatal("obsolete suffix selector remains in schema")
			}
			config := map[string]interface{}{"backend_environment_variable_strict_mode": test.config}
			prepared, diags := b.PrepareConfig(decodeBackendConfig(t, b, config))
			if test.wantError {
				if !diags.HasErrors() {
					t.Fatal("expected invalid boolean to fail")
				}
				return
			}
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if got := prepared.GetAttr("backend_environment_variable_strict_mode").True(); got != test.want {
				t.Fatalf("strict mode: got %t, want %t", got, test.want)
			}
			if test.want {
				if !prepared.GetAttr("client_id").IsNull() {
					t.Fatal("strict mode consumed a provider or obsolete environment variable")
				}
			} else if prepared.GetAttr("client_id").AsString() != "legacy-client" {
				t.Fatal("default mode did not use the generic fallback")
			}
		})
	}
}

func TestBackendDefaultEnvironmentFallback(t *testing.T) {
	clearBackendEnvironment(t)
	b := New().(*Backend)
	for attr, def := range b.SDKLikeDefaults {
		if attr == "backend_environment_variable_strict_mode" {
			continue
		}
		for _, env := range def.EnvVars {
			if strings.HasPrefix(env, "ARM_BACKEND_") {
				continue
			}
			value := "legacy-" + env
			if b.Schema.Attributes[attr].Type.Equals(cty.Bool) {
				value = "true"
			}
			t.Setenv(env, value)
		}
	}
	for _, strictMode := range []interface{}{nil, false} {
		raw := decodeBackendConfig(t, b, map[string]interface{}{
			"backend_environment_variable_strict_mode": strictMode,
			"client_id": "explicit-client",
			"use_cli":   false,
		})
		want, wantDiags := b.Base.PrepareConfig(raw)
		got, diags := b.PrepareConfig(raw)
		if diags.HasErrors() || wantDiags.HasErrors() {
			t.Fatalf("unexpected diagnostics: %v / %v", diags, wantDiags)
		}
		for attr := range b.Schema.Attributes {
			if !got.GetAttr(attr).RawEquals(want.GetAttr(attr)) {
				t.Errorf("legacy preparation differs for %s", attr)
			}
		}
	}
}

func TestBackendEnvironmentMappings(t *testing.T) {
	clearBackendEnvironment(t)
	b := New().(*Backend)
	for attr, def := range b.SDKLikeDefaults {
		if attr == "backend_environment_variable_strict_mode" {
			continue
		}
		t.Run(attr, func(t *testing.T) {
			ty := b.Schema.Attributes[attr].Type
			backendNames := map[string]bool{}
			seenFallback := false
			for _, env := range def.EnvVars {
				if strings.HasPrefix(env, "ARM_BACKEND_") {
					if seenFallback {
						t.Fatalf("backend alias appears after provider fallback: %s", env)
					}
					backendNames[env] = true
				} else {
					seenFallback = true
					if strings.HasPrefix(env, "ARM_") &&
						env != "ARM_METADATA_HOST" &&
						env != "ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID" &&
						!backendNames["ARM_BACKEND_"+strings.TrimPrefix(env, "ARM_")] {
						t.Fatalf("missing explicit ARM_BACKEND alternative for %s", env)
					}
				}
			}
			for _, strictMode := range []bool{false, true} {
				t.Run(map[bool]string{false: "default", true: "strict"}[strictMode], func(t *testing.T) {
					config := map[string]interface{}{"backend_environment_variable_strict_mode": strictMode}
					var allowed []string
					for _, env := range def.EnvVars {
						if !strictMode || strings.HasPrefix(env, "ARM_BACKEND_") ||
							env == "ACTIONS_ID_TOKEN_REQUEST_URL" ||
							env == "ACTIONS_ID_TOKEN_REQUEST_TOKEN" ||
							env == "SYSTEM_OIDCREQUESTURI" ||
							env == "SYSTEM_ACCESSTOKEN" {
							allowed = append(allowed, env)
						} else {
							t.Setenv(env, "ignored")
						}
					}
					want := cty.NullVal(ty)
					if def.Fallback != "" {
						want = cty.StringVal(def.Fallback)
						if ty.Equals(cty.Bool) {
							want = cty.BoolVal(def.Fallback == "true")
						}
					}
					if strictMode && attr == "use_cli" {
						want = cty.False
					}
					got := prepareBackendConfig(t, b, config).GetAttr(attr)
					if !got.RawEquals(want) {
						t.Fatalf("missing value: got %s, want %s", got.GoString(), want.GoString())
					}
					for first, name := range allowed {
						t.Run(name, func(t *testing.T) {
							for i, env := range allowed {
								value := "from-" + env
								if ty.Equals(cty.Bool) {
									value = "true"
								}
								if i < first {
									value = ""
								}
								t.Setenv(env, value)
							}
							want := cty.StringVal("from-" + name)
							if ty.Equals(cty.Bool) {
								want = cty.True
							}
							got := prepareBackendConfig(t, b, config).GetAttr(attr)
							if !got.RawEquals(want) {
								t.Fatalf("got %s, want %s", got.GoString(), want.GoString())
							}
							if ty.Equals(cty.Bool) {
								t.Setenv(name, "false")
								if !prepareBackendConfig(t, b, config).GetAttr(attr).RawEquals(cty.False) {
									t.Fatal("false environment value lost precedence")
								}
							}
							explicit := maps.Clone(config)
							if ty.Equals(cty.Bool) {
								explicit[attr] = false
								want = cty.False
							} else {
								explicit[attr] = "explicit"
								want = cty.StringVal("explicit")
							}
							got = prepareBackendConfig(t, b, explicit).GetAttr(attr)
							if !got.RawEquals(want) {
								t.Fatalf("explicit value lost precedence: got %s, want %s", got.GoString(), want.GoString())
							}
						})
					}
				})
			}
			if !reflect.DeepEqual(def, b.SDKLikeDefaults[attr]) {
				t.Fatal("PrepareConfig mutated canonical defaults")
			}
		})
	}
}

func TestBackendEnvironmentAliases(t *testing.T) {
	b := New().(*Backend)
	tests := map[string][]string{
		"metadata_host": {"ARM_BACKEND_METADATA_HOSTNAME", "ARM_METADATA_HOSTNAME", "ARM_METADATA_HOST"},
		"ado_pipeline_service_connection_id": {
			"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID",
			"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID",
			"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID",
			"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID",
		},
	}
	for attr, want := range tests {
		if got := b.SDKLikeDefaults[attr].EnvVars; !reflect.DeepEqual(got, want) {
			t.Errorf("%s aliases: got %v, want %v", attr, got, want)
		}
	}
}

func TestBackendCredentialPairs(t *testing.T) {
	pairs := []struct {
		direct, file, backendDirectEnv, backendFileEnv, directEnv, fileEnv string
		read                                                               func(*backendbase.SDKLikeData) (*string, error)
	}{
		{"client_id", "client_id_file_path", "ARM_BACKEND_CLIENT_ID", "ARM_BACKEND_CLIENT_ID_FILE_PATH", "ARM_CLIENT_ID", "ARM_CLIENT_ID_FILE_PATH", getClientId},
		{"client_secret", "client_secret_file_path", "ARM_BACKEND_CLIENT_SECRET", "ARM_BACKEND_CLIENT_SECRET_FILE_PATH", "ARM_CLIENT_SECRET", "ARM_CLIENT_SECRET_FILE_PATH", getClientSecret},
		{"oidc_token", "oidc_token_file_path", "ARM_BACKEND_OIDC_TOKEN", "ARM_BACKEND_OIDC_TOKEN_FILE_PATH", "ARM_OIDC_TOKEN", "ARM_OIDC_TOKEN_FILE_PATH", getOidcToken},
	}
	for _, pair := range pairs {
		t.Run(pair.direct, func(t *testing.T) {
			tests := []struct {
				name                         string
				strictMode                   bool
				configDirect, configFile     string
				selectedDirect, selectedFile string
				want                         string
				wantError                    bool
			}{
				{name: "explicit direct and backend file must match", strictMode: true, configDirect: "explicit", selectedFile: "selected", wantError: true},
				{name: "explicit file and backend direct must match", strictMode: true, configFile: "explicit", selectedDirect: "selected", wantError: true},
				{name: "matching explicit direct and backend file", strictMode: true, configDirect: "same", selectedFile: "same", want: "same"},
				{name: "matching explicit file and backend direct", strictMode: true, configFile: "same", selectedDirect: "same", want: "same"},
				{name: "matching explicit pair", strictMode: true, configDirect: "same", configFile: "same", want: "same"},
				{name: "mismatching explicit pair", strictMode: true, configDirect: "direct", configFile: "file", wantError: true},
				{name: "matching selected pair", strictMode: true, selectedDirect: "same", selectedFile: "same", want: "same"},
				{name: "mismatching selected pair", strictMode: true, selectedDirect: "direct", selectedFile: "file", wantError: true},
				{name: "selected file", strictMode: true, selectedFile: "file", want: "file"},
				{name: "legacy mixed-source mismatch unchanged", configDirect: "explicit", selectedFile: "environment", wantError: true},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					clearBackendEnvironment(t)
					config := map[string]interface{}{"backend_environment_variable_strict_mode": test.strictMode}
					if test.configDirect != "" {
						config[pair.direct] = test.configDirect
					}
					if test.configFile != "" {
						config[pair.file] = writeCredentialFile(t, test.configFile)
					}
					directEnv, fileEnv := pair.directEnv, pair.fileEnv
					if test.strictMode {
						directEnv, fileEnv = pair.backendDirectEnv, pair.backendFileEnv
					}
					t.Setenv(directEnv, test.selectedDirect)
					if test.selectedFile != "" {
						t.Setenv(fileEnv, writeCredentialFile(t, test.selectedFile))
					}
					if test.strictMode {
						t.Setenv(pair.directEnv, "unselected")
						t.Setenv(pair.fileEnv, "unselected-missing-file")
					}
					data := backendbase.NewSDKLikeData(prepareBackendConfig(t, New(), config))
					got, err := pair.read(&data)
					if test.wantError {
						if err == nil || !strings.Contains(err.Error(), "mismatch between supplied") {
							t.Fatalf("expected a mismatch error, got %v", err)
						}
					} else if err != nil {
						t.Fatal(err)
					} else if *got != test.want {
						t.Fatalf("got %q, want %q", *got, test.want)
					}
				})
			}
		})
	}
}

func TestBackendAuthenticationPriority(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]interface{}
		env    map[string]string
		want   auth.Authorizer
	}{
		{
			name:   "explicit assertion before request",
			config: map[string]interface{}{"oidc_token": "explicit-assertion"},
			env: map[string]string{
				"ARM_BACKEND_OIDC_REQUEST_URL":                 "https://example.invalid/request",
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "connection",
			},
			want: &auth.ClientAssertionAuthorizer{},
		},
		{
			name:   "environment assertion before explicit request",
			config: map[string]interface{}{"oidc_request_url": "https://example.invalid/request"},
			env:    map[string]string{"ARM_BACKEND_OIDC_TOKEN": "backend-assertion"},
			want:   &auth.ClientAssertionAuthorizer{},
		},
		{
			name:   "both explicit methods use SDK priority",
			config: map[string]interface{}{"oidc_token": "explicit-assertion", "oidc_request_url": "https://example.invalid/request"},
			want:   &auth.ClientAssertionAuthorizer{},
		},
		{
			name: "both environment methods use SDK priority",
			env: map[string]string{
				"ARM_BACKEND_OIDC_TOKEN":       "backend-assertion",
				"ARM_BACKEND_OIDC_REQUEST_URL": "https://example.invalid/request",
			},
			want: &auth.ClientAssertionAuthorizer{},
		},
		{
			name:   "AKS assertion before request",
			config: map[string]interface{}{"use_aks_workload_identity": true, "ado_pipeline_service_connection_id": "connection"},
			want:   &auth.ClientAssertionAuthorizer{},
		},
		{
			name:   "ADO request before GitHub request",
			config: map[string]interface{}{"ado_pipeline_service_connection_id": "connection"},
			env:    map[string]string{"ARM_BACKEND_OIDC_REQUEST_URL": "https://example.invalid/request"},
			want:   &auth.ADOPipelineOIDCAuthorizer{},
		},
		{
			name: "GitHub request",
			env:  map[string]string{"ARM_BACKEND_OIDC_REQUEST_URL": "https://example.invalid/request"},
			want: &auth.GitHubOIDCAuthorizer{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, strictMode := range []bool{false, true} {
				t.Run(map[bool]string{false: "default", true: "strict"}[strictMode], func(t *testing.T) {
					clearBackendEnvironment(t)
					t.Setenv("ARM_BACKEND_CLIENT_ID", "client")
					t.Setenv("ARM_BACKEND_TENANT_ID", "tenant")
					t.Setenv("ARM_BACKEND_OIDC_REQUEST_TOKEN", "request-token")
					t.Setenv("SYSTEM_OIDCREQUESTURI", "https://example.invalid/pipeline")
					t.Setenv("AZURE_CLIENT_ID", "client")
					t.Setenv("AZURE_TENANT_ID", "tenant")
					t.Setenv("AZURE_FEDERATED_TOKEN_FILE", writeCredentialFile(t, "aks-assertion"))
					for name, value := range test.env {
						t.Setenv(name, value)
					}
					config := map[string]interface{}{
						"backend_environment_variable_strict_mode": strictMode,
						"use_oidc":         true,
						"use_azuread_auth": true,
						"use_cli":          false,
					}
					maps.Copy(config, test.config)
					b := New().(*Backend)
					if diags := b.Configure(prepareBackendConfig(t, b, config)); diags.HasErrors() {
						t.Fatal(diags.ErrWithWarnings())
					}
					cached, ok := b.apiClient.azureAdStorageAuth.(*auth.CachedAuthorizer)
					if !ok || reflect.TypeOf(cached.Source) != reflect.TypeOf(test.want) {
						t.Fatalf("expected %T in cached authorizer, got %#v", test.want, b.apiClient.azureAdStorageAuth)
					}
				})
			}
		})
	}
}

func TestBackendStrictJobTokenDefaults(t *testing.T) {
	tests := []struct {
		name                               string
		config                             map[string]interface{}
		env                                map[string]string
		wantConnection, wantURL, wantToken string
	}{
		{name: "job inputs without connection", wantURL: "native-url", wantToken: "native-token"},
		{
			name:    "native task alias cannot select a connection",
			env:     map[string]string{"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID": "native-connection"},
			wantURL: "native-url", wantToken: "native-token",
		},
		{
			name: "GitHub job inputs retain precedence over SYSTEM inputs",
			env: map[string]string{
				"ACTIONS_ID_TOKEN_REQUEST_URL":   "github-url",
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "github-token",
			},
			wantURL: "github-url", wantToken: "github-token",
		},
		{
			name: "backend inputs override GitHub job inputs",
			env: map[string]string{
				"ACTIONS_ID_TOKEN_REQUEST_URL":   "github-url",
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "github-token",
				"ARM_BACKEND_OIDC_REQUEST_URL":   "backend-url",
				"ARM_BACKEND_OIDC_REQUEST_TOKEN": "backend-token",
			},
			wantURL: "backend-url", wantToken: "backend-token",
		},
		{
			name:   "explicit inputs override backend and GitHub inputs",
			config: map[string]interface{}{"oidc_request_url": "explicit-url", "oidc_request_token": "explicit-token"},
			env: map[string]string{
				"ACTIONS_ID_TOKEN_REQUEST_URL":   "github-url",
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "github-token",
				"ARM_BACKEND_OIDC_REQUEST_URL":   "backend-url",
				"ARM_BACKEND_OIDC_REQUEST_TOKEN": "backend-token",
			},
			wantURL: "explicit-url", wantToken: "explicit-token",
		},
		{
			name:           "explicit connection",
			config:         map[string]interface{}{"ado_pipeline_service_connection_id": "explicit-connection"},
			wantConnection: "explicit-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name:           "selected OIDC alias",
			env:            map[string]string{"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection"},
			wantConnection: "selected-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name: "missing job inputs remain unset",
			env: map[string]string{
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection",
				"SYSTEM_OIDCREQUESTURI":                        "",
				"SYSTEM_ACCESSTOKEN":                           "",
			},
			wantConnection: "selected-connection",
		},
		{
			name: "backend alias wins over generic canonical alias",
			env: map[string]string{
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID":       "provider-connection",
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "backend-connection",
			},
			wantConnection: "backend-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name: "selected ARM request overrides",
			env: map[string]string{
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection",
				"ARM_BACKEND_OIDC_REQUEST_URL":                 "selected-url",
				"ARM_BACKEND_OIDC_REQUEST_TOKEN":               "selected-token",
			},
			wantConnection: "selected-connection", wantURL: "selected-url", wantToken: "selected-token",
		},
		{
			name: "explicit request overrides",
			config: map[string]interface{}{
				"ado_pipeline_service_connection_id": "explicit-connection",
				"oidc_request_url":                   "explicit-url",
				"oidc_request_token":                 "explicit-token",
			},
			env: map[string]string{
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection",
				"ARM_BACKEND_OIDC_REQUEST_URL":                 "selected-url",
				"ARM_BACKEND_OIDC_REQUEST_TOKEN":               "selected-token",
			},
			wantConnection: "explicit-connection", wantURL: "explicit-url", wantToken: "explicit-token",
		},
		{
			name: "selected URL with native bearer",
			env: map[string]string{
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection",
				"ARM_BACKEND_OIDC_REQUEST_URL":                 "selected-url",
			},
			wantConnection: "selected-connection", wantURL: "selected-url", wantToken: "native-token",
		},
		{
			name: "native URL with selected bearer",
			env: map[string]string{
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection",
				"ARM_BACKEND_OIDC_REQUEST_TOKEN":               "selected-token",
			},
			wantConnection: "selected-connection", wantURL: "native-url", wantToken: "selected-token",
		},
		{
			name:   "empty overrides use native broker",
			config: map[string]interface{}{"oidc_request_url": "", "oidc_request_token": ""},
			env: map[string]string{
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection",
				"ARM_BACKEND_OIDC_REQUEST_URL":                 "",
				"ARM_BACKEND_OIDC_REQUEST_TOKEN":               "",
			},
			wantConnection: "selected-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name:           "explicit assertion does not discard connection defaults",
			config:         map[string]interface{}{"oidc_token": "explicit-assertion"},
			env:            map[string]string{"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection"},
			wantConnection: "selected-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name: "environment assertion and connection are left to SDK selection",
			env: map[string]string{
				"ARM_BACKEND_OIDC_TOKEN":                       "selected-assertion",
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "selected-connection",
			},
			wantConnection: "selected-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name:           "explicit assertion and connection are left to SDK selection",
			config:         map[string]interface{}{"oidc_token": "explicit-assertion", "ado_pipeline_service_connection_id": "explicit-connection"},
			wantConnection: "explicit-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name:           "blank connection does not discard job inputs",
			env:            map[string]string{"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": " "},
			wantConnection: " ", wantURL: "native-url", wantToken: "native-token",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			for _, name := range []string{
				"ARM_OIDC_REQUEST_URL", "ARM_OIDC_REQUEST_TOKEN",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID", "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID",
				"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID",
			} {
				t.Setenv(name, "unselected")
			}
			t.Setenv("SYSTEM_OIDCREQUESTURI", "native-url")
			t.Setenv("SYSTEM_ACCESSTOKEN", "native-token")
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			config := map[string]interface{}{"backend_environment_variable_strict_mode": true}
			maps.Copy(config, test.config)
			b := New()
			prepared, diags := b.PrepareConfig(decodeBackendConfig(t, b, config))
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			data := backendbase.NewSDKLikeData(prepared)
			for attr, want := range map[string]string{
				"ado_pipeline_service_connection_id": test.wantConnection,
				"oidc_request_url":                   test.wantURL,
				"oidc_request_token":                 test.wantToken,
			} {
				if got := data.String(attr); got != want {
					t.Errorf("%s: got %q, want %q", attr, got, want)
				}
			}
			repeated, diags := b.PrepareConfig(prepared)
			if diags.HasErrors() || !repeated.RawEquals(prepared) {
				t.Fatal("native broker defaults changed on repeated preparation")
			}
		})
	}
}

func TestBackendStrictADONativeBrokerAcquisition(t *testing.T) {
	clearBackendEnvironment(t)
	for name, value := range map[string]string{
		"ARM_BACKEND_CLIENT_ID":                        "backend-client",
		"ARM_BACKEND_TENANT_ID":                        "backend-tenant",
		"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "backend-connection",
		"ARM_BACKEND_USE_OIDC":                         "true",
		"ARM_BACKEND_USE_AZUREAD":                      "true",
		"ARM_CLIENT_ID":                                "provider-client",
		"ARM_TENANT_ID":                                "provider-tenant",
		"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID":         "provider-connection",
		"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID":      "provider-connection",
		"SYSTEM_OIDCREQUESTURI":                        "https://ado.example.invalid/oidc",
		"SYSTEM_ACCESSTOKEN":                           "job-bearer",
	} {
		t.Setenv(name, value)
	}
	originalClient := auth.Client
	t.Cleanup(func() { auth.Client = originalClient })
	requests := 0
	auth.Client = testAuthHTTPClient(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected auth request method: %s", req.Method)
		}
		var body string
		switch req.URL.Host {
		case "ado.example.invalid":
			if req.URL.Query().Get("serviceConnectionId") != "backend-connection" {
				t.Fatal("OIDC broker did not use the selected backend service connection")
			}
			if req.Header.Get("Authorization") != "Bearer job-bearer" {
				t.Fatal("OIDC broker did not use the native job bearer token")
			}
			body = `{"oidcToken":"issued-assertion"}`
		case "login.microsoftonline.com":
			if !strings.HasPrefix(req.URL.Path, "/backend-tenant/") {
				t.Fatalf("unexpected token exchange tenant: %s", req.URL.Path)
			}
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if req.PostForm.Get("client_id") != "backend-client" || req.PostForm.Get("client_assertion") != "issued-assertion" {
				t.Fatal("token exchange did not use the backend client and broker assertion")
			}
			body = `{"access_token":"storage-token","token_type":"Bearer","expires_in":3600}`
		default:
			t.Fatalf("unexpected auth request: %s", req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	b := New().(*Backend)
	prepared := prepareBackendConfig(t, b, map[string]interface{}{"backend_environment_variable_strict_mode": true})
	if diags := b.Configure(prepared); diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
	token, err := b.apiClient.azureAdStorageAuth.Token(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "storage-token" || requests != 2 {
		t.Fatalf("expected broker acquisition and token exchange, got %d requests", requests)
	}
}

type testAuthHTTPClient func(*http.Request) (*http.Response, error)

func (f testAuthHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestBackendStrictRequiresAuthOptIn(t *testing.T) {
	clearBackendEnvironment(t)
	for _, name := range []string{"ARM_USE_CLI", "ARM_USE_MSI", "ARM_USE_AKS_WORKLOAD_IDENTITY"} {
		t.Setenv(name, "true")
	}
	for _, name := range []string{
		"ARM_CLIENT_SECRET", "ARM_CLIENT_CERTIFICATE", "ARM_ACCESS_KEY", "ARM_SAS_TOKEN",
		"ACTIONS_ID_TOKEN_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_TOKEN",
		"SYSTEM_OIDCREQUESTURI", "SYSTEM_ACCESSTOKEN", "AZURESUBSCRIPTION_SERVICE_CONNECTION_ID",
	} {
		t.Setenv(name, "unselected")
	}
	t.Setenv("ARM_BACKEND_USE_OIDC", "true")
	t.Setenv("ARM_CLIENT_ID", "provider-client")
	t.Setenv("ARM_TENANT_ID", "provider-tenant")
	t.Setenv("AZURE_CLIENT_ID", "aks-client")
	t.Setenv("AZURE_TENANT_ID", "aks-tenant")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "unused-token-file")
	config := map[string]interface{}{"backend_environment_variable_strict_mode": true, "use_azuread_auth": true}
	b := New().(*Backend)
	prepared := prepareBackendConfig(t, b, config)
	for _, attr := range []string{"use_cli", "use_msi", "use_aks_workload_identity"} {
		if prepared.GetAttr(attr).True() {
			t.Fatalf("unsuffixed auth flag enabled %s", attr)
		}
	}
	diags := b.Configure(prepared)
	if !diags.HasErrors() || !strings.Contains(diags.Err().Error(), "no Authorizer could be configured") {
		t.Fatalf("expected missing selected credentials to fail without trying CLI/MSI/AKS, got %v", diags)
	}

	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "selected environment opt-in", true: "explicit opt-in"}[explicit], func(t *testing.T) {
			clearBackendEnvironment(t)
			t.Setenv("AZURE_FEDERATED_TOKEN_FILE", writeCredentialFile(t, "aks-assertion"))
			config := map[string]interface{}{"backend_environment_variable_strict_mode": true}
			if explicit {
				config["use_aks_workload_identity"] = true
			} else {
				t.Setenv("ARM_BACKEND_USE_AKS_WORKLOAD_IDENTITY", "true")
			}
			data := backendbase.NewSDKLikeData(prepareBackendConfig(t, New(), config))
			clientID, clientErr := getClientId(&data)
			tenantID, tenantErr := getTenantId(&data)
			token, tokenErr := getOidcToken(&data)
			if clientErr != nil || tenantErr != nil || tokenErr != nil {
				t.Fatalf("unexpected AKS errors: %v, %v, %v", clientErr, tenantErr, tokenErr)
			}
			if *clientID != "aks-client" || *tenantID != "aks-tenant" || *token != "aks-assertion" {
				t.Fatal("explicitly enabled AKS did not use its native identity inputs")
			}
		})
	}
}

func TestBackendInvalidBackendBoolean(t *testing.T) {
	clearBackendEnvironment(t)
	t.Setenv("ARM_USE_OIDC", "true")
	t.Setenv("ARM_BACKEND_USE_OIDC", "invalid")
	b := New()
	_, diags := b.PrepareConfig(decodeBackendConfig(t, b, nil))
	if !diags.HasErrors() || !strings.Contains(diags.Err().Error(), `invalid value for "use_oidc"`) {
		t.Fatalf("selected invalid flag must not fall back to the unsuffixed value: %v", diags)
	}
}

func TestBackendAuthorizers(t *testing.T) {
	tests := []struct {
		name       string
		strictMode bool
		env        map[string]string
		want       auth.Authorizer
	}{
		{"backend GitHub broker", true, map[string]string{"ARM_BACKEND_OIDC_REQUEST_URL": "https://example.invalid/github", "ARM_BACKEND_OIDC_REQUEST_TOKEN": "bearer"}, &auth.GitHubOIDCAuthorizer{}},
		{"strict GitHub job inputs", true, map[string]string{"ACTIONS_ID_TOKEN_REQUEST_URL": "https://example.invalid/github", "ACTIONS_ID_TOKEN_REQUEST_TOKEN": "bearer"}, &auth.GitHubOIDCAuthorizer{}},
		{"backend ADO broker", true, map[string]string{"ARM_BACKEND_OIDC_REQUEST_URL": "https://example.invalid/pipeline", "ARM_BACKEND_OIDC_REQUEST_TOKEN": "bearer", "ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "connection"}, &auth.ADOPipelineOIDCAuthorizer{}},
		{"backend assertion", true, map[string]string{"ARM_BACKEND_OIDC_TOKEN": "assertion"}, &auth.ClientAssertionAuthorizer{}},
		{"backend MSI opt-in", true, map[string]string{"ARM_BACKEND_USE_MSI": "true"}, &auth.ManagedIdentityAuthorizer{}},
		{"legacy GitHub broker", false, map[string]string{"ACTIONS_ID_TOKEN_REQUEST_URL": "https://example.invalid/github", "ACTIONS_ID_TOKEN_REQUEST_TOKEN": "bearer"}, &auth.GitHubOIDCAuthorizer{}},
		{"legacy ADO task alias", false, map[string]string{"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID": "connection", "SYSTEM_OIDCREQUESTURI": "https://example.invalid/pipeline", "SYSTEM_ACCESSTOKEN": "bearer"}, &auth.ADOPipelineOIDCAuthorizer{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			t.Setenv("ARM_BACKEND_CLIENT_ID", "client")
			t.Setenv("ARM_BACKEND_TENANT_ID", "tenant")
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			if test.strictMode {
				for _, name := range []string{"ARM_CLIENT_SECRET", "ARM_CLIENT_CERTIFICATE", "ARM_CLIENT_CERTIFICATE_PATH", "ARM_OIDC_TOKEN", "ARM_ACCESS_KEY", "ARM_SAS_TOKEN", "AZURESUBSCRIPTION_SERVICE_CONNECTION_ID"} {
					t.Setenv(name, "unselected")
				}
			}
			b := New().(*Backend)
			prepared := prepareBackendConfig(t, b, map[string]interface{}{"backend_environment_variable_strict_mode": test.strictMode, "use_oidc": true, "use_azuread_auth": true, "use_cli": false})
			if diags := b.Configure(prepared); diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			cached, ok := b.apiClient.azureAdStorageAuth.(*auth.CachedAuthorizer)
			if !ok || reflect.TypeOf(cached.Source) != reflect.TypeOf(test.want) {
				t.Fatalf("expected %T in cached authorizer, got %#v", test.want, b.apiClient.azureAdStorageAuth)
			}
		})
	}
}

func TestBackendMinimalMultitenantOIDC(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want auth.Authorizer
	}{
		{
			name: "GitHub",
			env: map[string]string{
				"ACTIONS_ID_TOKEN_REQUEST_URL":   "https://example.invalid/github",
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "job-bearer",
			},
			want: &auth.GitHubOIDCAuthorizer{},
		},
		{
			name: "Azure Pipelines",
			env: map[string]string{
				"ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID": "backend-connection",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID":         "provider-connection",
				"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID":      "provider-connection",
				"SYSTEM_OIDCREQUESTURI":                        "https://example.invalid/pipeline",
				"SYSTEM_ACCESSTOKEN":                           "job-bearer",
			},
			want: &auth.ADOPipelineOIDCAuthorizer{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			for name, value := range map[string]string{
				"ARM_CLIENT_ID":         "provider-client",
				"ARM_TENANT_ID":         "provider-tenant",
				"ARM_SUBSCRIPTION_ID":   "provider-subscription",
				"ARM_USE_OIDC":          "true",
				"ARM_USE_AZUREAD":       "true",
				"ARM_BACKEND_CLIENT_ID": "backend-client",
				"ARM_BACKEND_TENANT_ID": "backend-tenant",
			} {
				t.Setenv(name, value)
			}
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			b := New().(*Backend)
			prepared := prepareBackendConfig(t, b, nil)
			data := backendbase.NewSDKLikeData(prepared)
			if data.Bool("backend_environment_variable_strict_mode") || !data.Bool("use_oidc") || !data.Bool("use_azuread_auth") {
				t.Fatal("minimal config did not inherit shared OIDC/data-plane flags")
			}
			if data.String("client_id") != "backend-client" || data.String("tenant_id") != "backend-tenant" {
				t.Fatal("backend identity overrides were not used")
			}
			if data.String("subscription_id") != "provider-subscription" {
				t.Fatal("omitted backend subscription did not use the generic fallback")
			}
			if test.name == "Azure Pipelines" && data.String("ado_pipeline_service_connection_id") != "backend-connection" {
				t.Fatal("provider service connection took precedence over backend connection")
			}
			if diags := b.Configure(prepared); diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			cached, ok := b.apiClient.azureAdStorageAuth.(*auth.CachedAuthorizer)
			if !ok || reflect.TypeOf(cached.Source) != reflect.TypeOf(test.want) {
				t.Fatalf("expected %T in cached authorizer, got %#v", test.want, b.apiClient.azureAdStorageAuth)
			}
		})
	}
}

func TestBackendConcurrentPreparation(t *testing.T) {
	clearBackendEnvironment(t)
	t.Setenv("ARM_CLIENT_ID", "provider-client")
	t.Setenv("ARM_BACKEND_CLIENT_ID", "backend-client")
	t.Setenv("ARM_SUBSCRIPTION_ID", "provider-subscription")
	b := New().(*Backend)
	original := maps.Clone(b.SDKLikeDefaults)
	for attr, def := range original {
		def.EnvVars = append([]string(nil), def.EnvVars...)
		original[attr] = def
	}
	inputs := make([]cty.Value, 2)
	for i, strictMode := range []bool{false, true} {
		inputs[i] = decodeBackendConfig(t, b, map[string]interface{}{"backend_environment_variable_strict_mode": strictMode})
	}
	var wg sync.WaitGroup
	for range 20 {
		for i, wantSubscription := range []cty.Value{cty.StringVal("provider-subscription"), cty.NullVal(cty.String)} {
			wg.Go(func() {
				prepared, diags := b.PrepareConfig(inputs[i])
				if diags.HasErrors() {
					t.Error(diags.ErrWithWarnings())
					return
				}
				if got := prepared.GetAttr("client_id").AsString(); got != "backend-client" {
					t.Errorf("got %q, want backend-client", got)
				}
				if !prepared.GetAttr("subscription_id").RawEquals(wantSubscription) {
					t.Error("provider fallback crossed strict-mode boundaries")
				}
				repeated, diags := b.PrepareConfig(prepared)
				if diags.HasErrors() || !repeated.RawEquals(prepared) {
					t.Error("repeated preparation changed the result")
				}
			})
		}
	}
	wg.Wait()
	if !reflect.DeepEqual(original, b.SDKLikeDefaults) {
		t.Fatal("concurrent preparation mutated schema defaults")
	}
}

func prepareBackendConfig(t *testing.T, b backend.Backend, config map[string]interface{}) cty.Value {
	t.Helper()
	prepared, diags := b.PrepareConfig(decodeBackendConfig(t, b, config))
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
	return prepared
}

func decodeBackendConfig(t *testing.T, b backend.Backend, config map[string]interface{}) cty.Value {
	t.Helper()
	raw := map[string]interface{}{"storage_account_name": "testaccount", "container_name": "testcontainer", "key": "test.tfstate"}
	maps.Copy(raw, config)
	result, diags := hcldec.Decode(backend.TestWrapConfig(raw), b.ConfigSchema().DecoderSpec(), nil)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	return result
}

func clearBackendEnvironment(t *testing.T) {
	t.Helper()
	for _, def := range New().(*Backend).SDKLikeDefaults {
		for _, name := range def.EnvVars {
			t.Setenv(name, "")
		}
	}
}

func writeCredentialFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
