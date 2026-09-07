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

func TestBackendEnvironmentSuffixSelector(t *testing.T) {
	tests := []struct {
		name, control string
		config        interface{}
		want          string
		wantError     bool
	}{
		{name: "unset"},
		{name: "control environment", control: "_STATE_2", want: "_STATE_2"},
		{name: "explicit configuration wins", control: "_PROVIDER", config: "_BACKEND", want: "_BACKEND"},
		{name: "explicit empty disables control", control: "_BACKEND", config: ""},
		{name: "explicit empty disables invalid control", control: "invalid", config: ""},
		{name: "underscore only", config: "_", wantError: true},
		{name: "missing underscore", config: "STATE", wantError: true},
		{name: "lowercase", config: "_state", wantError: true},
		{name: "hyphen", config: "_STATE-2", wantError: true},
		{name: "whitespace", config: " _STATE", wantError: true},
		{name: "newline", config: "_STATE\n", wantError: true},
		{name: "non ASCII", config: "_\u00e9", wantError: true},
		{name: "invalid control", control: "invalid", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_SUFFIX", test.control)
			t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_SUFFIX_BACKEND", "_IGNORED")
			t.Setenv("ARM_CLIENT_ID", "legacy-client")
			t.Setenv("ARM_CLIENT_ID_BACKEND", "selected-client")
			b := New()
			config := map[string]interface{}{"environment_variable_suffix": test.config}
			prepared, diags := b.PrepareConfig(decodeBackendConfig(t, b, config))
			if test.wantError {
				if !diags.HasErrors() || !strings.Contains(diags.Err().Error(), "Invalid environment variable suffix") {
					t.Fatalf("expected suffix diagnostic, got %v", diags)
				}
				return
			}
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if got := prepared.GetAttr("environment_variable_suffix").AsString(); got != test.want {
				t.Fatalf("suffix: got %q, want %q", got, test.want)
			}
			if test.want == "" && prepared.GetAttr("client_id").AsString() != "legacy-client" {
				t.Fatal("empty suffix did not restore legacy defaults")
			}
		})
	}
}

func TestBackendEmptySuffixCompatibility(t *testing.T) {
	clearBackendEnvironment(t)
	b := New().(*Backend)
	for attr, def := range b.SDKLikeDefaults {
		if attr == "environment_variable_suffix" {
			continue
		}
		for _, env := range def.EnvVars {
			value := "legacy-" + env
			if b.Schema.Attributes[attr].Type.Equals(cty.Bool) {
				value = "true"
			}
			t.Setenv(env, value)
			t.Setenv(env+"_BACKEND", "must-not-be-read")
		}
	}
	for _, selector := range []interface{}{nil, ""} {
		raw := decodeBackendConfig(t, b, map[string]interface{}{
			"environment_variable_suffix": selector,
			"client_id":                   "explicit-client",
			"use_cli":                     false,
		})
		want, wantDiags := b.Base.PrepareConfig(raw)
		got, diags := b.PrepareConfig(raw)
		if diags.HasErrors() || wantDiags.HasErrors() {
			t.Fatalf("unexpected diagnostics: %v / %v", diags, wantDiags)
		}
		for attr := range b.Schema.Attributes {
			if attr != "environment_variable_suffix" && !got.GetAttr(attr).RawEquals(want.GetAttr(attr)) {
				t.Errorf("legacy preparation differs for %s", attr)
			}
		}
	}
}

func TestBackendSuffixMappings(t *testing.T) {
	clearBackendEnvironment(t)
	b := New().(*Backend)
	suffix := "_STATE_2"
	for attr, def := range b.SDKLikeDefaults {
		if attr == "environment_variable_suffix" {
			continue
		}
		t.Run(attr, func(t *testing.T) {
			config := map[string]interface{}{"environment_variable_suffix": suffix}
			ty := b.Schema.Attributes[attr].Type
			for _, env := range def.EnvVars {
				// Invalid boolean defaults and bogus credential files must be ignored in strict mode.
				t.Setenv(env, "ambient")
				if !strings.HasPrefix(env, "ARM_") {
					t.Setenv(env+suffix, "ambient-native-suffixed")
				}
			}
			want := cty.NullVal(ty)
			if def.Fallback != "" {
				want = cty.StringVal(def.Fallback)
				if ty.Equals(cty.Bool) {
					want = cty.BoolVal(def.Fallback == "true")
				}
			}
			if attr == "use_cli" {
				want = cty.False
			}
			got := prepareBackendConfig(t, b, config).GetAttr(attr)
			if !got.RawEquals(want) {
				t.Fatalf("ambient fallback for %s: got %s, want %s", attr, got.GoString(), want.GoString())
			}

			for first, name := range def.EnvVars {
				if !strings.HasPrefix(name, "ARM_") {
					continue
				}
				t.Run(name, func(t *testing.T) {
					for i, env := range def.EnvVars {
						if !strings.HasPrefix(env, "ARM_") {
							continue
						}
						value := "selected-" + env
						if ty.Equals(cty.Bool) {
							value = "true"
						}
						if i < first {
							value = ""
						}
						t.Setenv(env+suffix, value)
					}
					want := cty.StringVal("selected-" + name)
					if ty.Equals(cty.Bool) {
						want = cty.True
					}
					got := prepareBackendConfig(t, b, config).GetAttr(attr)
					if !got.RawEquals(want) {
						t.Fatalf("got %s, want %s", got.GoString(), want.GoString())
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
			if !reflect.DeepEqual(def, b.SDKLikeDefaults[attr]) {
				t.Fatal("PrepareConfig mutated canonical defaults")
			}
		})
	}
}

func TestBackendSuffixCredentialPairs(t *testing.T) {
	pairs := []struct {
		direct, file, directEnv, fileEnv string
		read                             func(*backendbase.SDKLikeData) (*string, error)
	}{
		{"client_id", "client_id_file_path", "ARM_CLIENT_ID", "ARM_CLIENT_ID_FILE_PATH", getClientId},
		{"client_secret", "client_secret_file_path", "ARM_CLIENT_SECRET", "ARM_CLIENT_SECRET_FILE_PATH", getClientSecret},
		{"oidc_token", "oidc_token_file_path", "ARM_OIDC_TOKEN", "ARM_OIDC_TOKEN_FILE_PATH", getOidcToken},
	}
	for _, pair := range pairs {
		t.Run(pair.direct, func(t *testing.T) {
			tests := []struct {
				name                         string
				suffix                       string
				configDirect, configFile     string
				selectedDirect, selectedFile string
				want                         string
				wantError                    bool
			}{
				{name: "explicit direct beats selected file", suffix: "_STATE", configDirect: "explicit", selectedFile: "selected", want: "explicit"},
				{name: "explicit file beats selected direct", suffix: "_STATE", configFile: "explicit", selectedDirect: "selected", want: "explicit"},
				{name: "matching explicit pair", suffix: "_STATE", configDirect: "same", configFile: "same", want: "same"},
				{name: "mismatching explicit pair", suffix: "_STATE", configDirect: "direct", configFile: "file", wantError: true},
				{name: "matching selected pair", suffix: "_STATE", selectedDirect: "same", selectedFile: "same", want: "same"},
				{name: "mismatching selected pair", suffix: "_STATE", selectedDirect: "direct", selectedFile: "file", wantError: true},
				{name: "selected file", suffix: "_STATE", selectedFile: "file", want: "file"},
				{name: "legacy mixed-source mismatch unchanged", configDirect: "explicit", selectedFile: "environment", wantError: true},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					clearBackendEnvironment(t)
					config := map[string]interface{}{"environment_variable_suffix": test.suffix}
					if test.configDirect != "" {
						config[pair.direct] = test.configDirect
					}
					if test.configFile != "" {
						config[pair.file] = writeCredentialFile(t, test.configFile)
					}
					t.Setenv(pair.directEnv+test.suffix, test.selectedDirect)
					if test.selectedFile != "" {
						t.Setenv(pair.fileEnv+test.suffix, writeCredentialFile(t, test.selectedFile))
					}
					if test.suffix != "" {
						t.Setenv(pair.directEnv, "ambient")
						t.Setenv(pair.fileEnv, "ambient-missing-file")
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

func TestBackendSuffixOIDCMethods(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]interface{}
		env       map[string]string
		wantError bool
		want      map[string]string
	}{
		{
			name:   "explicit assertion wins",
			config: map[string]interface{}{"oidc_token": "explicit-assertion"},
			env:    map[string]string{"ARM_OIDC_REQUEST_URL_STATE": "selected-url", "ARM_OIDC_REQUEST_TOKEN_STATE": "selected-bearer", "ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID_STATE": "selected-connection"},
			want:   map[string]string{"oidc_token": "explicit-assertion", "oidc_request_url": "", "oidc_request_token": "", "ado_pipeline_service_connection_id": ""},
		},
		{
			name:   "explicit request wins",
			config: map[string]interface{}{"oidc_request_url": "explicit-url"},
			env:    map[string]string{"ARM_OIDC_TOKEN_STATE": "selected-assertion", "ARM_OIDC_TOKEN_FILE_PATH_STATE": "selected-file", "ARM_OIDC_REQUEST_TOKEN_STATE": "selected-bearer", "ARM_USE_AKS_WORKLOAD_IDENTITY_STATE": "true"},
			want:   map[string]string{"oidc_token": "", "oidc_token_file_path": "", "oidc_request_url": "explicit-url", "oidc_request_token": "selected-bearer"},
		},
		{
			name:      "explicit methods conflict",
			config:    map[string]interface{}{"oidc_token": "explicit-assertion", "oidc_request_url": "explicit-url"},
			wantError: true,
		},
		{
			name:      "selected methods conflict",
			env:       map[string]string{"ARM_OIDC_TOKEN_STATE": "selected-assertion", "ARM_OIDC_REQUEST_URL_STATE": "selected-url"},
			wantError: true,
		},
		{
			name:      "AKS and request conflict",
			config:    map[string]interface{}{"use_aks_workload_identity": true, "ado_pipeline_service_connection_id": "explicit-connection"},
			wantError: true,
		},
		{
			name:   "legacy competing inputs unchanged",
			config: map[string]interface{}{"environment_variable_suffix": "", "oidc_token": "assertion", "oidc_request_url": "request-url"},
			want:   map[string]string{"oidc_token": "assertion", "oidc_request_url": "request-url"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			config := map[string]interface{}{"environment_variable_suffix": "_STATE"}
			maps.Copy(config, test.config)
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			b := New()
			prepared, diags := b.PrepareConfig(decodeBackendConfig(t, b, config))
			if test.wantError {
				if !diags.HasErrors() || !strings.Contains(diags.Err().Error(), "Conflicting OIDC authentication settings") {
					t.Fatalf("expected conflict diagnostic, got %v", diags)
				}
				return
			}
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			data := backendbase.NewSDKLikeData(prepared)
			for attr, want := range test.want {
				if got := data.String(attr); got != want {
					t.Errorf("%s: got %q, want %q", attr, got, want)
				}
			}
		})
	}
}

func TestBackendSuffixADONativeBrokerDefaults(t *testing.T) {
	tests := []struct {
		name                               string
		config                             map[string]interface{}
		env                                map[string]string
		wantConnection, wantURL, wantToken string
		wantConflict                       bool
	}{
		{name: "no selected connection"},
		{
			name: "native task alias cannot select a connection",
			env:  map[string]string{"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID_STATE": "native-suffixed-connection"},
		},
		{
			name:           "explicit connection",
			config:         map[string]interface{}{"ado_pipeline_service_connection_id": "explicit-connection"},
			wantConnection: "explicit-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name:           "selected OIDC alias",
			env:            map[string]string{"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "selected-connection"},
			wantConnection: "selected-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name: "selected connection does not enable GitHub fallback",
			env: map[string]string{
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "selected-connection",
				"SYSTEM_OIDCREQUESTURI":                      "",
				"SYSTEM_ACCESSTOKEN":                         "",
			},
			wantConnection: "selected-connection",
		},
		{
			name: "canonical selected alias retains precedence",
			env: map[string]string{
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID_STATE": "canonical-connection",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE":   "alias-connection",
			},
			wantConnection: "canonical-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name: "selected ARM request overrides",
			env: map[string]string{
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "selected-connection",
				"ARM_OIDC_REQUEST_URL_STATE":                 "selected-url",
				"ARM_OIDC_REQUEST_TOKEN_STATE":               "selected-token",
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
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "selected-connection",
				"ARM_OIDC_REQUEST_URL_STATE":                 "selected-url",
				"ARM_OIDC_REQUEST_TOKEN_STATE":               "selected-token",
			},
			wantConnection: "explicit-connection", wantURL: "explicit-url", wantToken: "explicit-token",
		},
		{
			name: "selected URL with native bearer",
			env: map[string]string{
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "selected-connection",
				"ARM_OIDC_REQUEST_URL_STATE":                 "selected-url",
			},
			wantConnection: "selected-connection", wantURL: "selected-url", wantToken: "native-token",
		},
		{
			name: "native URL with selected bearer",
			env: map[string]string{
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "selected-connection",
				"ARM_OIDC_REQUEST_TOKEN_STATE":               "selected-token",
			},
			wantConnection: "selected-connection", wantURL: "native-url", wantToken: "selected-token",
		},
		{
			name:   "empty overrides use native broker",
			config: map[string]interface{}{"oidc_request_url": "", "oidc_request_token": ""},
			env: map[string]string{
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "selected-connection",
				"ARM_OIDC_REQUEST_URL_STATE":                 "",
				"ARM_OIDC_REQUEST_TOKEN_STATE":               "",
			},
			wantConnection: "selected-connection", wantURL: "native-url", wantToken: "native-token",
		},
		{
			name:   "explicit assertion suppresses selected connection and native broker",
			config: map[string]interface{}{"oidc_token": "explicit-assertion"},
			env:    map[string]string{"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "ignored-connection"},
		},
		{
			name: "selected assertion and connection still conflict",
			env: map[string]string{
				"ARM_OIDC_TOKEN_STATE":                       "selected-assertion",
				"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "selected-connection",
			},
			wantConflict: true,
		},
		{
			name:         "explicit assertion and connection still conflict",
			config:       map[string]interface{}{"oidc_token": "explicit-assertion", "ado_pipeline_service_connection_id": "explicit-connection"},
			wantConflict: true,
		},
		{
			name:           "blank connection does not unlock native broker",
			env:            map[string]string{"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": " "},
			wantConnection: " ",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			for _, name := range []string{
				"ARM_OIDC_REQUEST_URL", "ARM_OIDC_REQUEST_TOKEN",
				"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID", "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID",
				"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID",
				"ACTIONS_ID_TOKEN_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_TOKEN",
			} {
				t.Setenv(name, "ambient")
			}
			t.Setenv("SYSTEM_OIDCREQUESTURI", "native-url")
			t.Setenv("SYSTEM_ACCESSTOKEN", "native-token")
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			config := map[string]interface{}{"environment_variable_suffix": "_STATE"}
			maps.Copy(config, test.config)
			b := New()
			prepared, diags := b.PrepareConfig(decodeBackendConfig(t, b, config))
			if test.wantConflict {
				if !diags.HasErrors() || !strings.Contains(diags.Err().Error(), "Conflicting OIDC authentication settings") {
					t.Fatalf("expected OIDC conflict, got %v", diags)
				}
				return
			}
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

func TestBackendSuffixADONativeBrokerAcquisition(t *testing.T) {
	clearBackendEnvironment(t)
	for name, value := range map[string]string{
		"ARM_CLIENT_ID_BACKEND":                        "backend-client",
		"ARM_TENANT_ID_BACKEND":                        "backend-tenant",
		"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_BACKEND": "backend-connection",
		"ARM_USE_OIDC_BACKEND":                         "true",
		"ARM_USE_AZUREAD_BACKEND":                      "true",
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
	prepared := prepareBackendConfig(t, b, map[string]interface{}{"environment_variable_suffix": "_BACKEND"})
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

func TestBackendSuffixAmbientAuth(t *testing.T) {
	clearBackendEnvironment(t)
	for _, name := range []string{"ARM_USE_CLI", "ARM_USE_MSI", "ARM_USE_AKS_WORKLOAD_IDENTITY"} {
		t.Setenv(name, "true")
	}
	for _, name := range []string{
		"ARM_CLIENT_SECRET", "ARM_CLIENT_CERTIFICATE", "ARM_ACCESS_KEY", "ARM_SAS_TOKEN",
		"ACTIONS_ID_TOKEN_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_TOKEN",
		"SYSTEM_OIDCREQUESTURI", "SYSTEM_ACCESSTOKEN", "AZURESUBSCRIPTION_SERVICE_CONNECTION_ID",
	} {
		t.Setenv(name, "ambient")
	}
	t.Setenv("ARM_USE_OIDC_STATE", "true")
	t.Setenv("ARM_CLIENT_ID_STATE", "selected-client")
	t.Setenv("ARM_TENANT_ID_STATE", "selected-tenant")
	t.Setenv("AZURE_CLIENT_ID", "ambient-client")
	t.Setenv("AZURE_TENANT_ID", "ambient-tenant")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "ambient-missing-file")
	config := map[string]interface{}{"environment_variable_suffix": "_STATE", "use_azuread_auth": true}
	b := New().(*Backend)
	prepared := prepareBackendConfig(t, b, config)
	for _, attr := range []string{"use_cli", "use_msi", "use_aks_workload_identity"} {
		if prepared.GetAttr(attr).True() {
			t.Fatalf("ambient auth enabled %s", attr)
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
			config := map[string]interface{}{"environment_variable_suffix": "_STATE"}
			if explicit {
				config["use_aks_workload_identity"] = true
			} else {
				t.Setenv("ARM_USE_AKS_WORKLOAD_IDENTITY_STATE", "true")
			}
			data := backendbase.NewSDKLikeData(prepareBackendConfig(t, New(), config))
			clientID, clientErr := getClientId(&data)
			tenantID, tenantErr := getTenantId(&data)
			token, tokenErr := getOidcToken(&data)
			if clientErr != nil || tenantErr != nil || tokenErr != nil {
				t.Fatalf("unexpected AKS errors: %v, %v, %v", clientErr, tenantErr, tokenErr)
			}
			if *clientID != "ambient-client" || *tenantID != "ambient-tenant" || *token != "aks-assertion" {
				t.Fatal("explicitly enabled AKS did not use its native identity inputs")
			}
		})
	}
}

func TestBackendSuffixInvalidSelectedBoolean(t *testing.T) {
	clearBackendEnvironment(t)
	t.Setenv("ARM_USE_OIDC", "true")
	t.Setenv("ARM_USE_OIDC_STATE", "invalid")
	b := New()
	_, diags := b.PrepareConfig(decodeBackendConfig(t, b, map[string]interface{}{"environment_variable_suffix": "_STATE"}))
	if !diags.HasErrors() || !strings.Contains(diags.Err().Error(), `invalid value for "use_oidc"`) {
		t.Fatalf("selected invalid flag must not fall back to the unsuffixed value: %v", diags)
	}
}

func TestBackendSuffixAuthorizers(t *testing.T) {
	tests := []struct {
		name, suffix string
		env          map[string]string
		want         auth.Authorizer
	}{
		{"selected GitHub broker", "_STATE", map[string]string{"ARM_OIDC_REQUEST_URL_STATE": "https://example.invalid/github", "ARM_OIDC_REQUEST_TOKEN_STATE": "bearer"}, &auth.GitHubOIDCAuthorizer{}},
		{"selected ADO broker", "_STATE", map[string]string{"ARM_OIDC_REQUEST_URL_STATE": "https://example.invalid/pipeline", "ARM_OIDC_REQUEST_TOKEN_STATE": "bearer", "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE": "connection"}, &auth.ADOPipelineOIDCAuthorizer{}},
		{"selected assertion", "_STATE", map[string]string{"ARM_OIDC_TOKEN_STATE": "assertion"}, &auth.ClientAssertionAuthorizer{}},
		{"selected MSI opt-in", "_STATE", map[string]string{"ARM_USE_MSI_STATE": "true"}, &auth.ManagedIdentityAuthorizer{}},
		{"legacy GitHub broker", "", map[string]string{"ACTIONS_ID_TOKEN_REQUEST_URL": "https://example.invalid/github", "ACTIONS_ID_TOKEN_REQUEST_TOKEN": "bearer"}, &auth.GitHubOIDCAuthorizer{}},
		{"legacy ADO task alias", "", map[string]string{"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID": "connection", "SYSTEM_OIDCREQUESTURI": "https://example.invalid/pipeline", "SYSTEM_ACCESSTOKEN": "bearer"}, &auth.ADOPipelineOIDCAuthorizer{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearBackendEnvironment(t)
			t.Setenv("ARM_CLIENT_ID"+test.suffix, "client")
			t.Setenv("ARM_TENANT_ID"+test.suffix, "tenant")
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			if test.suffix != "" {
				for _, name := range []string{"ARM_CLIENT_SECRET", "ARM_CLIENT_CERTIFICATE", "ARM_CLIENT_CERTIFICATE_PATH", "ARM_OIDC_TOKEN", "ARM_ACCESS_KEY", "ARM_SAS_TOKEN", "AZURESUBSCRIPTION_SERVICE_CONNECTION_ID"} {
					t.Setenv(name, "ambient")
				}
			}
			b := New().(*Backend)
			prepared := prepareBackendConfig(t, b, map[string]interface{}{"environment_variable_suffix": test.suffix, "use_oidc": true, "use_azuread_auth": true, "use_cli": false})
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

func TestBackendSuffixConcurrentPreparation(t *testing.T) {
	clearBackendEnvironment(t)
	t.Setenv("ARM_CLIENT_ID", "legacy")
	t.Setenv("ARM_CLIENT_ID_ONE", "one")
	t.Setenv("ARM_CLIENT_ID_TWO", "two")
	b := New().(*Backend)
	original := maps.Clone(b.SDKLikeDefaults)
	inputs := make([]cty.Value, 3)
	for i, suffix := range []string{"", "_ONE", "_TWO"} {
		inputs[i] = decodeBackendConfig(t, b, map[string]interface{}{"environment_variable_suffix": suffix})
	}
	var wg sync.WaitGroup
	for range 20 {
		for i, want := range []string{"legacy", "one", "two"} {
			wg.Go(func() {
				prepared, diags := b.PrepareConfig(inputs[i])
				if diags.HasErrors() {
					t.Error(diags.ErrWithWarnings())
					return
				}
				if got := prepared.GetAttr("client_id").AsString(); got != want {
					t.Errorf("got %q, want %q", got, want)
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
			for _, suffix := range []string{"_BACKEND", "_STATE", "_STATE_2", "_ONE", "_TWO"} {
				t.Setenv(name+suffix, "")
			}
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
