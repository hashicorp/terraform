package rpcapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestServerOpenCloseConfig(t *testing.T) {
	ctx := context.Background()
	configDir := t.TempDir()

	// We're not actually going to use any providers here, so we can
	// provide a nil factory without any problems.
	client := newV1ClientForTests(t, configDir, coreOptsWithTestProvider(nil))

	resp, err := client.OpenConfigCwd(ctx, &tfcore1.OpenConfigCwd_Request{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Diagnostics) > 0 {
		t.Fatalf("unexpected diagnostics\n%s", cmp.Diff(nil, resp.Diagnostics))
	}
	if resp.ConfigId == 0 {
		t.Fatal("not assigned a configuration id")
	}
	t.Logf("configuration id is %d", resp.ConfigId)

	_, err = client.CloseConfig(ctx, &tfcore1.CloseConfig_Request{
		ConfigId: resp.ConfigId,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerValidateConfig(t *testing.T) {
	ctx := context.Background()
	configDir := t.TempDir()
	configFile := filepath.Join(configDir, "main.tf")

	err := os.WriteFile(configFile, []byte(`
		resource "test" "thing" {
			# The mock provider always fails validation
		}
	`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	client := newV1ClientForTests(t, configDir, coreOptsWithTestProvider(func() (providers.Interface, error) {
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Expected Error",
			Detail:   "This error is expected",
			Subject: &hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 1, Column: 2, Byte: 3},
				End:      hcl.Pos{Line: 4, Column: 5, Byte: 6},
			},
		})
		return &terraform.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				ResourceTypes: map[string]providers.Schema{
					"test": {
						Block: &configschema.Block{},
					},
				},
			},
			ValidateResourceConfigResponse: &providers.ValidateResourceConfigResponse{
				Diagnostics: diags,
			},
		}, nil
	}))

	configResp, err := client.OpenConfigCwd(ctx, &tfcore1.OpenConfigCwd_Request{})
	if err != nil {
		t.Fatal(err)
	}
	if len(configResp.Diagnostics) > 0 {
		// We're not expecting diagnostics until validation time
		t.Fatalf("unexpected diagnostics\n%s", cmp.Diff(nil, configResp.Diagnostics))
	}
	if configResp.ConfigId == 0 {
		t.Fatal("not assigned a configuration id")
	}
	configID := configResp.ConfigId
	t.Logf("configuration id is %d", configID)

	got, err := client.ValidateConfig(ctx, &tfcore1.ValidateConfig_Request{
		ConfigId: configID,
	})
	want := &tfcore1.ValidateConfig_Response{
		Diagnostics: []*tfcore1.Diagnostic{
			{
				Severity: tfcore1.Diagnostic_ERROR,
				Summary:  "Expected Error",
				Detail:   "This error is expected",
				Subject: &tfcore1.SourceRange{
					Filename: "main.tf",
					Start: &tfcore1.SourceRange_Pos{
						Line: 1, Column: 2, Byte: 3,
					},
					End: &tfcore1.SourceRange_Pos{
						Line: 4, Column: 5, Byte: 6,
					},
				},
			},
		},
	}
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got, protoCmpOpt); diff != "" {
		t.Fatalf("wrong response from ValidateConfig\n%s", diff)
	}

	_, err = client.CloseConfig(ctx, &tfcore1.CloseConfig_Request{
		ConfigId: configID,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.ValidateConfig(ctx, &tfcore1.ValidateConfig_Request{
		ConfigId: configID,
	})
	if err == nil {
		t.Errorf("validation still succeeded after closing the configuration")
	}

}

func TestServerCreatePlanInitial(t *testing.T) {
	ctx := context.Background()
	configDir := t.TempDir()
	configFile := filepath.Join(configDir, "main.tf")

	err := os.WriteFile(configFile, []byte(`
		resource "test" "thing" {
		}

		output "a" {
			value = "boop"
		}

		output "b" {
			value     = "beep"
			sensitive = true
		}
	`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	client := newV1ClientForTests(t, configDir, coreOptsWithTestProvider(func() (providers.Interface, error) {
		return &terraform.MockProvider{
			GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
				ResourceTypes: map[string]providers.Schema{
					"test": {
						Block: &configschema.Block{},
					},
				},
			},
			ValidateResourceConfigResponse: &providers.ValidateResourceConfigResponse{},
			UpgradeResourceStateResponse: &providers.UpgradeResourceStateResponse{
				UpgradedState: cty.EmptyObjectVal,
			},
			PlanResourceChangeResponse: &providers.PlanResourceChangeResponse{
				PlannedState: cty.EmptyObjectVal,
			},
		}, nil
	}))

	configResp, err := client.OpenConfigCwd(ctx, &tfcore1.OpenConfigCwd_Request{})
	if err != nil {
		t.Fatal(err)
	}
	if len(configResp.Diagnostics) > 0 {
		// We're not expecting diagnostics until validation time
		t.Fatalf("unexpected diagnostics\n%s", cmp.Diff(nil, configResp.Diagnostics))
	}
	if configResp.ConfigId == 0 {
		t.Fatal("not assigned a configuration id")
	}
	configID := configResp.ConfigId
	t.Logf("configuration id is %d", configID)

	got, err := client.CreatePlan(ctx, &tfcore1.CreatePlan_Request{
		ConfigId:     configID,
		PrevRunState: nil, // is if this is the first plan
		Options: &tfcore1.PlanOptions{
			Mode: tfcore1.PlanOptions_NORMAL,
		},
	})
	want := &tfcore1.CreatePlan_Response{
		PlanId: 1,
		PlannedOutputValues: map[string]*tfcore1.DynamicValue{
			"a": {
				TypeJson:     []byte(`"string"`),
				ValueMsgpack: []byte("\xa4boop"),
			},
			"b": {
				TypeJson:     []byte(`"string"`),
				ValueMsgpack: []byte("\xa4beep"),
				Sensitive:    true,
			},
		},
	}
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got, protoCmpOpt); diff != "" {
		t.Fatalf("wrong response from CreatePlan\n%s", diff)
	}

	_, err = client.CloseConfig(ctx, &tfcore1.CloseConfig_Request{
		ConfigId: configID,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.CreatePlan(ctx, &tfcore1.CreatePlan_Request{
		ConfigId:     configID,
		PrevRunState: nil,
		Options: &tfcore1.PlanOptions{
			Mode: tfcore1.PlanOptions_NORMAL,
		},
	})
	if err == nil {
		t.Errorf("planning still succeeded after closing the configuration")
	}

}
