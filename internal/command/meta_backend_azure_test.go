// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
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
		name, suffix, requestTokenEnv, requestURLEnv string
	}{
		{"generic", "", "ARM_OIDC_REQUEST_TOKEN", "ARM_OIDC_REQUEST_URL"},
		{"backend", "_BACKEND", "ARM_OIDC_REQUEST_TOKEN_BACKEND", "ARM_OIDC_REQUEST_URL_BACKEND"},
		{"GitHub", "", "ACTIONS_ID_TOKEN_REQUEST_TOKEN", "ACTIONS_ID_TOKEN_REQUEST_URL"},
		{"Azure Pipelines", "", "SYSTEM_ACCESSTOKEN", "SYSTEM_OIDCREQUESTURI"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, def := range azure.New().(*azure.Backend).SDKLikeDefaults {
				for _, name := range def.EnvVars {
					t.Setenv(name, "")
				}
			}
			tokenFile := filepath.Join(t.TempDir(), "oidc-token")
			if err := os.WriteFile(tokenFile, []byte("plan-assertion"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("ARM_OIDC_TOKEN"+test.suffix, "plan-assertion")
			t.Setenv("ARM_OIDC_TOKEN_FILE_PATH"+test.suffix, tokenFile)
			t.Setenv(test.requestTokenEnv, "plan-bearer")
			t.Setenv(test.requestURLEnv, "https://example.invalid/oidc")
			if test.name == "Azure Pipelines" {
				t.Setenv("ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID", "service-connection")
			}

			file, parseDiags := hclsyntax.ParseConfig([]byte(`
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
			if got := prepared.GetAttr("oidc_token"); !got.RawEquals(cty.StringVal("plan-assertion")) {
				t.Fatalf("environment assertion was not prepared: %s", got.GoString())
			}
			if got := prepared.GetAttr("oidc_request_token"); !got.RawEquals(cty.StringVal("plan-bearer")) {
				t.Fatalf("environment request bearer was not prepared: %s", got.GoString())
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
			for _, secret := range []string{"plan-assertion", "plan-bearer"} {
				if bytes.Contains(saved.ConfigRaw, []byte(secret)) {
					t.Errorf("saved backend configuration contains %s", secret)
				}
			}
			if !planConfig.GetAttr("client_id").RawEquals(cty.StringVal("configured-client")) {
				t.Fatal("explicit backend identity was not retained in the plan")
			}

			t.Setenv("ARM_OIDC_TOKEN"+test.suffix, "apply-assertion")
			t.Setenv(test.requestTokenEnv, "apply-bearer")
			if err := os.WriteFile(tokenFile, []byte("apply-assertion"), 0600); err != nil {
				t.Fatal(err)
			}
			applyBackend := azure.New()
			applyConfig, diags := applyBackend.PrepareConfig(planConfig)
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if !applyConfig.GetAttr("oidc_token").RawEquals(cty.StringVal("apply-assertion")) ||
				!applyConfig.GetAttr("oidc_request_token").RawEquals(cty.StringVal("apply-bearer")) {
				t.Fatal("saved plan did not use fresh environment credentials")
			}
			if diags := applyBackend.Configure(applyConfig); diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
		})
	}
}
