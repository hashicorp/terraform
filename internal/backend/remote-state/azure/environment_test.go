// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
)

func TestBackendEnvironmentPrecedence(t *testing.T) {
	tests := []struct {
		attr string
		envs []string
	}{
		{"subscription_id", []string{"ARM_SUBSCRIPTION_ID_BACKEND", "ARM_SUBSCRIPTION_ID"}},
		{"tenant_id", []string{"ARM_TENANT_ID_BACKEND", "ARM_TENANT_ID"}},
		{"client_id", []string{"ARM_CLIENT_ID_BACKEND", "ARM_CLIENT_ID"}},
		{"client_id_file_path", []string{"ARM_CLIENT_ID_FILE_PATH_BACKEND", "ARM_CLIENT_ID_FILE_PATH"}},
		{"use_oidc", []string{"ARM_USE_OIDC_BACKEND", "ARM_USE_OIDC"}},
		{"use_azuread_auth", []string{"ARM_USE_AZUREAD_BACKEND", "ARM_USE_AZUREAD"}},
		{"oidc_token", []string{"ARM_OIDC_TOKEN_BACKEND", "ARM_OIDC_TOKEN"}},
		{"oidc_token_file_path", []string{"ARM_OIDC_TOKEN_FILE_PATH_BACKEND", "ARM_OIDC_TOKEN_FILE_PATH"}},
		{"oidc_request_url", []string{"ARM_OIDC_REQUEST_URL_BACKEND", "ARM_OIDC_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_URL", "SYSTEM_OIDCREQUESTURI"}},
		{"oidc_request_token", []string{"ARM_OIDC_REQUEST_TOKEN_BACKEND", "ARM_OIDC_REQUEST_TOKEN", "ACTIONS_ID_TOKEN_REQUEST_TOKEN", "SYSTEM_ACCESSTOKEN"}},
		{"ado_pipeline_service_connection_id", []string{"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID_BACKEND", "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_BACKEND", "ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID", "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID", "AZURESUBSCRIPTION_SERVICE_CONNECTION_ID"}},
	}

	for _, test := range tests {
		t.Run(test.attr, func(t *testing.T) {
			clearBackendEnvironment(t)
			b := New().(*Backend)
			attr := b.ConfigSchema().Attributes[test.attr]
			if attr == nil || !attr.Optional {
				t.Fatalf("%s must remain an optional schema attribute", test.attr)
			}
			if got := b.SDKLikeDefaults[test.attr].EnvVars; !reflect.DeepEqual(got, test.envs) {
				t.Fatalf("schema defaults: got %v, want %v", got, test.envs)
			}
			config := map[string]interface{}{}
			if strings.HasPrefix(test.attr, "oidc_request_") {
				config["ado_pipeline_service_connection_id"] = "service-connection"
			}

			for first, name := range test.envs {
				t.Run(name, func(t *testing.T) {
					for i, env := range test.envs {
						value := env
						if attr.Type.Equals(cty.Bool) {
							value = "true"
							if i == first {
								value = "false"
							}
						}
						if i < first {
							value = ""
						}
						t.Setenv(env, value)
					}
					want := cty.StringVal(name)
					if attr.Type.Equals(cty.Bool) {
						want = cty.False
					}
					got := prepareBackendConfig(t, b, config).GetAttr(test.attr)
					if !got.RawEquals(want) {
						t.Fatalf("got %s, want %s", got.GoString(), want.GoString())
					}
				})
			}

			t.Run("explicit configuration", func(t *testing.T) {
				for _, env := range test.envs {
					value := "environment"
					if attr.Type.Equals(cty.Bool) {
						value = "true"
					}
					t.Setenv(env, value)
				}
				var explicit interface{} = "configuration"
				want := cty.StringVal("configuration")
				if attr.Type.Equals(cty.Bool) {
					explicit, want = false, cty.False
				}
				explicitConfig := maps.Clone(config)
				explicitConfig[test.attr] = explicit
				got := prepareBackendConfig(t, b, explicitConfig).GetAttr(test.attr)
				if !got.RawEquals(want) {
					t.Fatalf("got %s, want %s", got.GoString(), want.GoString())
				}
			})

			t.Run("empty configuration uses environment", func(t *testing.T) {
				if !attr.Type.Equals(cty.String) {
					return
				}
				t.Setenv(test.envs[0], "environment")
				emptyConfig := maps.Clone(config)
				emptyConfig[test.attr] = ""
				got := prepareBackendConfig(t, b, emptyConfig).GetAttr(test.attr)
				if !got.RawEquals(cty.StringVal("environment")) {
					t.Fatalf("got %s, want environment fallback", got.GoString())
				}
			})

			t.Run("unset environment", func(t *testing.T) {
				for _, env := range test.envs {
					if err := os.Unsetenv(env); err != nil {
						t.Fatal(err)
					}
				}
				want := cty.NullVal(cty.String)
				if attr.Type.Equals(cty.Bool) {
					want = cty.False
				}
				got := prepareBackendConfig(t, b, config).GetAttr(test.attr)
				if !got.RawEquals(want) {
					t.Fatalf("got %s, want %s", got.GoString(), want.GoString())
				}
			})
			if attr.Type.Equals(cty.Bool) {
				for _, env := range test.envs {
					t.Run(env+"=true", func(t *testing.T) {
						t.Setenv(env, "true")
						got := prepareBackendConfig(t, b, config).GetAttr(test.attr)
						if !got.RawEquals(cty.True) {
							t.Fatalf("got %s, want true", got.GoString())
						}
					})
				}
			}
			if got := b.SDKLikeDefaults[test.attr].EnvVars; !reflect.DeepEqual(got, test.envs) {
				t.Fatalf("PrepareConfig mutated schema defaults: %v", got)
			}
		})
	}
}

func TestBackendCredentialPairPrecedence(t *testing.T) {
	pairs := []struct {
		direct, file, directEnv, fileEnv string
		read                             func(*backendbase.SDKLikeData) (*string, error)
	}{
		{"client_id", "client_id_file_path", "ARM_CLIENT_ID", "ARM_CLIENT_ID_FILE_PATH", getClientId},
		{"oidc_token", "oidc_token_file_path", "ARM_OIDC_TOKEN", "ARM_OIDC_TOKEN_FILE_PATH", getOidcToken},
	}
	for _, pair := range pairs {
		t.Run(pair.direct, func(t *testing.T) {
			tests := []struct {
				name                       string
				configDirect, configFile   string
				backendDirect, backendFile string
				genericDirect, genericFile string
				want                       string
				wantError                  bool
			}{
				{name: "config direct overrides environment file", configDirect: "config", backendFile: "backend", genericFile: "generic", want: "config"},
				{name: "config file overrides environment direct", configFile: "config", backendDirect: "backend", genericDirect: "generic", want: "config"},
				{name: "backend direct overrides generic file", backendDirect: "backend", genericFile: "generic", want: "backend"},
				{name: "backend file overrides generic direct", backendFile: "backend", genericDirect: "generic", want: "backend"},
				{name: "generic direct fallback", genericDirect: "generic", want: "generic"},
				{name: "generic file fallback", genericFile: "generic", want: "generic"},
				{name: "matching config pair", configDirect: "config", configFile: "config", want: "config"},
				{name: "mismatching config pair", configDirect: "direct", configFile: "file", wantError: true},
				{name: "matching backend pair", backendDirect: "backend", backendFile: "backend", want: "backend"},
				{name: "mismatching backend pair", backendDirect: "direct", backendFile: "file", wantError: true},
				{name: "matching generic pair", genericDirect: "generic", genericFile: "generic", want: "generic"},
				{name: "mismatching generic pair", genericDirect: "direct", genericFile: "file", wantError: true},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					clearBackendEnvironment(t)
					config := map[string]interface{}{}
					if test.configDirect != "" {
						config[pair.direct] = test.configDirect
					}
					if test.configFile != "" {
						config[pair.file] = writeCredentialFile(t, test.configFile)
					}
					t.Setenv(pair.directEnv+"_BACKEND", test.backendDirect)
					t.Setenv(pair.directEnv, test.genericDirect)
					if test.backendFile != "" {
						t.Setenv(pair.fileEnv+"_BACKEND", writeCredentialFile(t, test.backendFile))
					}
					if test.genericFile != "" {
						t.Setenv(pair.fileEnv, writeCredentialFile(t, test.genericFile))
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

func TestBackendOIDCSourceSelection(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]interface{}
		env    map[string]string
		want   map[string]string
	}{
		{
			name: "separate backend client retains original service connection fallback",
			env: map[string]string{
				"ARM_CLIENT_ID_BACKEND":                "backend-client",
				"ARM_CLIENT_ID":                        "provider-client",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID": "service-connection",
				"SYSTEM_OIDCREQUESTURI":                "pipeline-url",
				"SYSTEM_ACCESSTOKEN":                   "pipeline-token",
			},
			want: map[string]string{
				"client_id":                          "backend-client",
				"ado_pipeline_service_connection_id": "service-connection",
				"oidc_request_url":                   "pipeline-url",
				"oidc_request_token":                 "pipeline-token",
			},
		},
		{
			name:   "explicit client and request retain generic service connection",
			config: map[string]interface{}{"client_id": "backend-client", "oidc_request_url": "pipeline-url"},
			env:    map[string]string{"ARM_CLIENT_ID": "provider-client", "ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID": "service-connection"},
			want:   map[string]string{"client_id": "backend-client", "ado_pipeline_service_connection_id": "service-connection"},
		},
		{
			name: "ignored backend request default does not disable generic service connection",
			config: map[string]interface{}{
				"oidc_request_url": "explicit-url",
			},
			env: map[string]string{
				"ARM_OIDC_REQUEST_URL_BACKEND":           "ignored-url",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID": "service-connection",
			},
			want: map[string]string{
				"oidc_request_url":                   "explicit-url",
				"ado_pipeline_service_connection_id": "service-connection",
			},
		},
		{
			name: "backend assertion isolates provider service connection",
			env: map[string]string{
				"ARM_OIDC_TOKEN_BACKEND":                  "backend-token",
				"ARM_OIDC_TOKEN":                          "provider-token",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID":    "provider-connection",
				"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID": "task-connection",
				"SYSTEM_OIDCREQUESTURI":                   "pipeline-url",
				"SYSTEM_ACCESSTOKEN":                      "pipeline-token",
			},
			want: map[string]string{
				"oidc_token":                         "backend-token",
				"ado_pipeline_service_connection_id": "",
				"oidc_request_url":                   "",
				"oidc_request_token":                 "",
			},
		},
		{
			name: "backend request overrides provider assertion and service connection",
			env: map[string]string{
				"ARM_OIDC_REQUEST_URL_BACKEND":         "backend-url",
				"ARM_OIDC_REQUEST_TOKEN_BACKEND":       "backend-bearer",
				"ARM_OIDC_TOKEN":                       "provider-token",
				"ARM_OIDC_TOKEN_FILE_PATH":             "provider-file",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID": "provider-connection",
			},
			want: map[string]string{
				"oidc_request_url":                   "backend-url",
				"oidc_request_token":                 "backend-bearer",
				"oidc_token":                         "",
				"oidc_token_file_path":               "",
				"ado_pipeline_service_connection_id": "",
			},
		},
		{
			name:   "explicit service connection wins over isolation",
			config: map[string]interface{}{"ado_pipeline_service_connection_id": "explicit-connection"},
			env:    map[string]string{"ARM_OIDC_REQUEST_URL_BACKEND": "backend-url", "ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID": "provider-connection"},
			want:   map[string]string{"ado_pipeline_service_connection_id": "explicit-connection"},
		},
		{
			name: "backend service connection wins over isolation",
			env: map[string]string{
				"ARM_OIDC_REQUEST_TOKEN_BACKEND":               "backend-bearer",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_BACKEND": "backend-connection",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID":       "provider-connection",
				"SYSTEM_OIDCREQUESTURI":                        "pipeline-url",
			},
			want: map[string]string{"ado_pipeline_service_connection_id": "backend-connection", "oidc_request_url": "pipeline-url"},
		},
		{
			name: "raw pipelines request without service connection is not a GitHub request",
			env:  map[string]string{"SYSTEM_OIDCREQUESTURI": "pipeline-url", "SYSTEM_ACCESSTOKEN": "pipeline-token"},
			want: map[string]string{"oidc_request_url": "", "oidc_request_token": ""},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			data := backendbase.NewSDKLikeData(prepareBackendConfig(t, New(), test.config))
			for attr, want := range test.want {
				if got := data.String(attr); got != want {
					t.Errorf("%s: got %q, want %q", attr, got, want)
				}
			}
		})
	}
}

func TestBackendOIDCAuthorizerSelection(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want auth.Authorizer
	}{
		{
			name: "github refresh",
			env: map[string]string{
				"ACTIONS_ID_TOKEN_REQUEST_URL":   "https://example.invalid/github",
				"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "github-bearer",
			},
			want: &auth.GitHubOIDCAuthorizer{},
		},
		{
			name: "pipeline refresh",
			env: map[string]string{
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID": "connection",
				"SYSTEM_OIDCREQUESTURI":                "https://example.invalid/pipeline",
				"SYSTEM_ACCESSTOKEN":                   "pipeline-bearer",
			},
			want: &auth.ADOPipelineOIDCAuthorizer{},
		},
		{
			name: "backend request isolation",
			env: map[string]string{
				"ARM_OIDC_REQUEST_URL_BACKEND":         "https://example.invalid/github",
				"ARM_OIDC_REQUEST_TOKEN_BACKEND":       "backend-bearer",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID": "provider-connection",
				"ARM_OIDC_TOKEN":                       "provider-token",
			},
			want: &auth.GitHubOIDCAuthorizer{},
		},
		{
			name: "pipeline assertion without service connection",
			env: map[string]string{
				"ARM_OIDC_TOKEN_BACKEND":               "backend-assertion",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID": "provider-connection",
				"SYSTEM_OIDCREQUESTURI":                "https://example.invalid/pipeline",
				"SYSTEM_ACCESSTOKEN":                   "pipeline-bearer",
			},
			want: &auth.ClientAssertionAuthorizer{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			t.Setenv("ARM_CLIENT_ID_BACKEND", "backend-client")
			t.Setenv("ARM_TENANT_ID_BACKEND", "backend-tenant")
			t.Setenv("ARM_CLIENT_ID", "provider-client")
			t.Setenv("ARM_TENANT_ID", "provider-tenant")
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			b := New().(*Backend)
			prepared := prepareBackendConfig(t, b, map[string]interface{}{"use_oidc": true, "use_azuread_auth": true, "use_cli": false})
			if diags := b.Configure(prepared); diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			cached, ok := b.apiClient.azureAdStorageAuth.(*auth.CachedAuthorizer)
			if !ok {
				t.Fatalf("expected cached authorizer, got %T", b.apiClient.azureAdStorageAuth)
			}
			if reflect.TypeOf(cached.Source) != reflect.TypeOf(test.want) {
				t.Fatalf("got %T, want %T", cached.Source, test.want)
			}
		})
	}
}

func TestBackendEnvironmentInvalidBoolean(t *testing.T) {
	for _, name := range []string{"ARM_USE_OIDC_BACKEND", "ARM_USE_AZUREAD_BACKEND"} {
		t.Run(name, func(t *testing.T) {
			clearBackendEnvironment(t)
			t.Setenv(name, "invalid")
			t.Setenv(strings.TrimSuffix(name, "_BACKEND"), "true")
			b := New()
			config := decodeBackendConfig(t, b, nil)
			if _, diags := b.PrepareConfig(config); !diags.HasErrors() {
				t.Fatal("expected invalid backend boolean to fail instead of using generic fallback")
			}
		})
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
	raw := map[string]interface{}{
		"storage_account_name": "testaccount",
		"container_name":       "testcontainer",
		"key":                  "test.tfstate",
	}
	for name, value := range config {
		raw[name] = value
	}
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
