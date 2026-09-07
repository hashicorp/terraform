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
	tests := []struct {
		name, suffix, control, nativeRequestToken     string
		configSelector                                interface{}
		serviceConnectionEnv, configServiceConnection string
		request                                       bool
	}{
		{name: "legacy assertion"},
		{name: "explicit selector", suffix: "_STATE", configSelector: "_STATE"},
		{name: "control selector", suffix: "_BACKEND", control: "_BACKEND"},
		{name: "explicit empty selector", configSelector: "", control: "_BACKEND"},
		{name: "selected broker", suffix: "_STATE", configSelector: "_STATE", request: true},
		{name: "legacy GitHub broker", nativeRequestToken: "ACTIONS_ID_TOKEN_REQUEST_TOKEN", request: true},
		{name: "legacy Azure Pipelines broker", nativeRequestToken: "SYSTEM_ACCESSTOKEN", request: true},
		{name: "selected ADO native broker", suffix: "_STATE", configSelector: "_STATE", nativeRequestToken: "SYSTEM_ACCESSTOKEN", serviceConnectionEnv: "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_STATE", request: true},
		{name: "explicit ADO native broker", suffix: "_STATE", configSelector: "_STATE", nativeRequestToken: "SYSTEM_ACCESSTOKEN", configServiceConnection: "backend-connection", request: true},
		{name: "control-selected ADO native broker", suffix: "_BACKEND", control: "_BACKEND", nativeRequestToken: "SYSTEM_ACCESSTOKEN", serviceConnectionEnv: "ARM_OIDC_AZURE_SERVICE_CONNECTION_ID_BACKEND", request: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, def := range azure.New().(*azure.Backend).SDKLikeDefaults {
				for _, name := range def.EnvVars {
					t.Setenv(name, "")
					t.Setenv(name+"_STATE", "")
					t.Setenv(name+"_BACKEND", "")
				}
			}
			t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_SUFFIX", test.control)
			if test.suffix != "" {
				t.Setenv("ARM_OIDC_TOKEN", "ambient-assertion")
				t.Setenv("ARM_OIDC_REQUEST_TOKEN", "ambient-bearer")
				t.Setenv("ARM_OIDC_REQUEST_URL", "https://example.invalid/ambient-oidc")
			}
			tokenEnv := "ARM_OIDC_TOKEN" + test.suffix
			requestURLEnv := ""
			attr := "oidc_token"
			tokenFile := filepath.Join(t.TempDir(), "oidc-token")
			if test.request {
				attr = "oidc_request_token"
				tokenEnv = "ARM_OIDC_REQUEST_TOKEN" + test.suffix
				requestURLEnv = "ARM_OIDC_REQUEST_URL" + test.suffix
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
				t.Setenv("ARM_OIDC_TOKEN_FILE_PATH"+test.suffix, tokenFile)
			}
			t.Setenv(tokenEnv, "plan-credential")
			if test.serviceConnectionEnv != "" {
				t.Setenv(test.serviceConnectionEnv, "backend-connection")
			}

			selectorConfig := ""
			if test.configSelector != nil {
				selectorConfig = fmt.Sprintf("environment_variable_suffix = %q\n", test.configSelector)
			}
			if test.configServiceConnection != "" {
				selectorConfig += fmt.Sprintf("ado_pipeline_service_connection_id = %q\n", test.configServiceConnection)
			}
			file, parseDiags := hclsyntax.ParseConfig([]byte(selectorConfig+`
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
			if !prepared.GetAttr(attr).RawEquals(cty.StringVal("plan-credential")) {
				t.Fatalf("environment credential was not prepared for %s", attr)
			}
			if test.request && !prepared.GetAttr("oidc_request_url").RawEquals(cty.StringVal("https://example.invalid/oidc")) {
				t.Fatal("broker URL did not come from the selected or permitted native environment")
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
			wantSelector := cty.NullVal(cty.String)
			if test.configSelector != nil {
				wantSelector = cty.StringVal(test.configSelector.(string))
			}
			if !planConfig.GetAttr("environment_variable_suffix").RawEquals(wantSelector) {
				t.Fatal("selector persistence differs from explicit configuration")
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
			} else {
				if err := os.WriteFile(tokenFile, []byte("apply-credential"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if test.configSelector != nil {
				t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_SUFFIX", "_DIFFERENT")
			}
			applyBackend := azure.New()
			applyConfig, diags := applyBackend.PrepareConfig(planConfig)
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if !applyConfig.GetAttr(attr).RawEquals(cty.StringVal("apply-credential")) {
				t.Fatal("saved plan did not use fresh credentials from the selected environment")
			}
			if test.request && !applyConfig.GetAttr("oidc_request_url").RawEquals(cty.StringVal("https://example.invalid/apply-oidc")) {
				t.Fatal("saved plan did not use the fresh broker URL")
			}
			if diags := applyBackend.Configure(applyConfig); diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if test.control != "" && test.configSelector == nil {
				t.Setenv("ARM_BACKEND_ENVIRONMENT_VARIABLE_SUFFIX", "")
				withoutControl, diags := azure.New().PrepareConfig(planConfig)
				if diags.HasErrors() {
					t.Fatal(diags.ErrWithWarnings())
				}
				if withoutControl.GetAttr("environment_variable_suffix").AsString() != "" {
					t.Fatal("environment-derived selector unexpectedly survived in the saved plan")
				}
			}
		})
	}
}
