// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend/remote-state/azure"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/configs"
)

func TestMetaBackendAzureEnvironmentCredentialsNotPersisted(t *testing.T) {
	enabled, disabled := true, false
	tests := []struct {
		name, control, nativeRequestToken             string
		configStrict                                  *bool
		backendCredentials, request                   bool
		serviceConnectionEnv, configServiceConnection string
	}{
		{name: "generic assertion"},
		{name: "backend assertion", backendCredentials: true},
		{name: "strict backend assertion", configStrict: &enabled, backendCredentials: true},
		{name: "strict control environment", control: "true", backendCredentials: true},
		{name: "explicit false overrides control", configStrict: &disabled, control: "true", backendCredentials: true},
		{name: "backend broker", backendCredentials: true, request: true},
		{name: "strict backend broker", configStrict: &enabled, backendCredentials: true, request: true},
		{name: "GitHub broker fallback", nativeRequestToken: "ACTIONS_ID_TOKEN_REQUEST_TOKEN", request: true},
		{name: "strict GitHub job inputs", configStrict: &enabled, nativeRequestToken: "ACTIONS_ID_TOKEN_REQUEST_TOKEN", request: true},
		{name: "control-enabled GitHub job inputs", control: "true", nativeRequestToken: "ACTIONS_ID_TOKEN_REQUEST_TOKEN", request: true},
		{name: "ADO broker fallback", nativeRequestToken: "SYSTEM_ACCESSTOKEN", request: true},
		{name: "backend ADO native broker", nativeRequestToken: "SYSTEM_ACCESSTOKEN", serviceConnectionEnv: "ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID", request: true},
		{name: "strict backend ADO native broker", configStrict: &enabled, nativeRequestToken: "SYSTEM_ACCESSTOKEN", serviceConnectionEnv: "ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID", request: true},
		{name: "strict explicit ADO native broker", configStrict: &enabled, nativeRequestToken: "SYSTEM_ACCESSTOKEN", configServiceConnection: "backend-connection", request: true},
		{name: "control-enabled ADO native broker", control: "true", nativeRequestToken: "SYSTEM_ACCESSTOKEN", serviceConnectionEnv: "ARM_BACKEND_OIDC_AZURE_SERVICE_CONNECTION_ID", request: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, def := range azure.New().(*azure.Backend).SDKLikeDefaults {
				for _, name := range def.EnvVars {
					t.Setenv(name, "")
				}
			}
			t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_STRICT_MODE", test.control)
			wantStrict := test.control == "true"
			if test.configStrict != nil {
				wantStrict = *test.configStrict
			}
			if wantStrict {
				t.Setenv("ARM_OIDC_TOKEN", "provider-assertion")
				t.Setenv("ARM_OIDC_REQUEST_TOKEN", "provider-bearer")
				t.Setenv("ARM_OIDC_REQUEST_URL", "https://example.invalid/provider-oidc")
			}
			tokenEnv, tokenFileEnv := "ARM_OIDC_TOKEN", "ARM_OIDC_TOKEN_FILE_PATH"
			if test.backendCredentials {
				tokenEnv, tokenFileEnv = "ARM_BACKEND_OIDC_TOKEN", "ARM_BACKEND_OIDC_TOKEN_FILE_PATH"
			}
			requestURLEnv := ""
			attr := "oidc_token"
			tokenFile := filepath.Join(t.TempDir(), "oidc-token")
			if test.request {
				attr = "oidc_request_token"
				tokenEnv, requestURLEnv = "ARM_OIDC_REQUEST_TOKEN", "ARM_OIDC_REQUEST_URL"
				if test.backendCredentials {
					tokenEnv, requestURLEnv = "ARM_BACKEND_OIDC_REQUEST_TOKEN", "ARM_BACKEND_OIDC_REQUEST_URL"
				}
				switch test.nativeRequestToken {
				case "ACTIONS_ID_TOKEN_REQUEST_TOKEN":
					tokenEnv = test.nativeRequestToken
					requestURLEnv = "ACTIONS_ID_TOKEN_REQUEST_URL"
				case "SYSTEM_ACCESSTOKEN":
					tokenEnv = test.nativeRequestToken
					requestURLEnv = "SYSTEM_OIDCREQUESTURI"
					t.Setenv("AZURESUBSCRIPTION_SERVICE_CONNECTION_ID", "service-connection")
				}
				t.Setenv(requestURLEnv, "https://example.invalid/oidc")
			} else {
				if err := os.WriteFile(tokenFile, []byte("plan-credential"), 0600); err != nil {
					t.Fatal(err)
				}
				t.Setenv(tokenFileEnv, tokenFile)
			}
			t.Setenv(tokenEnv, "plan-credential")
			if test.serviceConnectionEnv != "" {
				t.Setenv(test.serviceConnectionEnv, "backend-connection")
			}

			settings := ""
			if test.configStrict != nil {
				settings = fmt.Sprintf("backend_environment_variable_strict_mode = %t\n", *test.configStrict)
			}
			if test.configServiceConnection != "" {
				settings += fmt.Sprintf("ado_pipeline_service_connection_id = %q\n", test.configServiceConnection)
			}
			file, parseDiags := hclsyntax.ParseConfig([]byte(settings+`
storage_account_name = "testaccount"
container_name       = "testcontainer"
key                  = "test.tfstate"
access_key           = "QUNDRVNTX0tFWQ0K"
client_id            = "configured-client"
`), "backend.tf", hcl.InitialPos)
			if parseDiags.HasErrors() {
				t.Fatal(parseDiags)
			}
			m := testMetaBackend(t, nil)
			b, rawConfig, diags := m.backendInitFromConfig(&configs.Backend{Type: "azurerm", Config: file.Body})
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			prepared, diags := b.PrepareConfig(rawConfig)
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if prepared.GetAttr("backend_environment_variable_strict_mode").True() != wantStrict {
				t.Fatal("incorrect strict-mode setting")
			}
			if !prepared.GetAttr(attr).RawEquals(cty.StringVal("plan-credential")) {
				t.Fatalf("environment credential was not prepared for %s", attr)
			}
			if test.request && !prepared.GetAttr("oidc_request_url").RawEquals(cty.StringVal("https://example.invalid/oidc")) {
				t.Fatal("broker URL did not come from the expected environment")
			}
			if (test.serviceConnectionEnv != "" || test.configServiceConnection != "") &&
				!prepared.GetAttr("ado_pipeline_service_connection_id").RawEquals(cty.StringVal("backend-connection")) {
				t.Fatal("backend service connection was not selected")
			}

			saved := &workdir.BackendConfigState{Type: "azurerm"}
			if err := saved.SetConfig(rawConfig, b.ConfigSchema()); err != nil {
				t.Fatal(err)
			}
			planBackend, err := saved.PlanData(b.ConfigSchema(), nil, "default")
			if err != nil {
				t.Fatal(err)
			}
			planConfig, err := planBackend.Config.Decode(b.ConfigSchema().ImpliedType())
			if err != nil {
				t.Fatal(err)
			}
			for _, attr := range []string{"oidc_token", "oidc_token_file_path", "oidc_request_token", "oidc_request_url"} {
				if !rawConfig.GetAttr(attr).IsNull() || !planConfig.GetAttr(attr).IsNull() {
					t.Errorf("environment default for %s was persisted", attr)
				}
			}
			if bytes.Contains(saved.ConfigRaw, []byte("plan-credential")) {
				t.Fatal("saved backend configuration contains the environment credential")
			}
			wantMode := cty.NullVal(cty.Bool)
			if test.configStrict != nil {
				wantMode = cty.BoolVal(*test.configStrict)
			}
			if !planConfig.GetAttr("backend_environment_variable_strict_mode").RawEquals(wantMode) {
				t.Fatal("strict-mode persistence differs from explicit configuration")
			}
			if !planConfig.GetAttr("client_id").RawEquals(cty.StringVal("configured-client")) {
				t.Fatal("explicit identity was not retained in the plan")
			}
			wantConnection := cty.NullVal(cty.String)
			if test.configServiceConnection != "" {
				wantConnection = cty.StringVal(test.configServiceConnection)
			}
			if !planConfig.GetAttr("ado_pipeline_service_connection_id").RawEquals(wantConnection) {
				t.Fatal("service connection persistence differs from explicit configuration")
			}

			t.Setenv(tokenEnv, "apply-credential")
			if test.request {
				t.Setenv(requestURLEnv, "https://example.invalid/apply-oidc")
			} else if err := os.WriteFile(tokenFile, []byte("apply-credential"), 0600); err != nil {
				t.Fatal(err)
			}
			if test.configStrict != nil {
				t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_STRICT_MODE", fmt.Sprint(!*test.configStrict))
			}
			applyBackend := azure.New()
			applyConfig, diags := applyBackend.PrepareConfig(planConfig)
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if applyConfig.GetAttr("backend_environment_variable_strict_mode").True() != wantStrict {
				t.Fatal("saved plan did not retain the expected strict-mode setting")
			}
			if !applyConfig.GetAttr(attr).RawEquals(cty.StringVal("apply-credential")) {
				t.Fatal("saved plan did not use fresh environment credentials")
			}
			if test.request && !applyConfig.GetAttr("oidc_request_url").RawEquals(cty.StringVal("https://example.invalid/apply-oidc")) {
				t.Fatal("saved plan did not use the fresh broker URL")
			}
			if diags := applyBackend.Configure(applyConfig); diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if test.control == "true" && test.configStrict == nil {
				t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_STRICT_MODE", "")
				withoutControl, diags := azure.New().PrepareConfig(planConfig)
				if diags.HasErrors() {
					t.Fatal(diags.ErrWithWarnings())
				}
				if withoutControl.GetAttr("backend_environment_variable_strict_mode").True() {
					t.Fatal("environment-derived strict mode unexpectedly survived in the saved plan")
				}
			}
		})
	}
}
