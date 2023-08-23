package stackruntime

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"
)

func TestPlanWithSingleResource(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "with-single-resource")

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	req := PlanRequest{
		Config: cfg,
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &req, &resp)
	gotChanges, diags := collectPlanOutput(changesCh, diagsCh)

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics\n%s", diags.ErrWithWarnings().Error())
	}

	// The order of emission for our planned changes is unspecified since it
	// depends on how the various goroutines get scheduled, and so we'll
	// arbitrarily sort gotChanges lexically by the name of the change type
	// so that we have some dependable order to diff against below.
	sort.Slice(gotChanges, func(i, j int) bool {
		ic := gotChanges[i]
		jc := gotChanges[j]
		return fmt.Sprintf("%T", ic) < fmt.Sprintf("%T", jc)
	})

	wantChanges := []stackplan.PlannedChange{
		&stackplan.PlannedChangeApplyable{
			Applyable: true,
		},
		&stackplan.PlannedChangeComponentInstance{
			Addr: stackaddrs.Absolute(
				stackaddrs.RootStackInstance,
				stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{Name: "self"},
				},
			),
			Action: plans.Create,
		},
		&stackplan.PlannedChangeHeader{
			TerraformVersion: version.SemVer,
		},
		&stackplan.PlannedChangeOutputValue{
			Addr:     stackaddrs.OutputValue{Name: "obj"},
			Action:   plans.Create,
			OldValue: mustPlanDynamicValue(cty.NullVal(cty.DynamicPseudoType)),
			NewValue: mustPlanDynamicValue(cty.ObjectVal(map[string]cty.Value{
				"input":  cty.StringVal("hello"),
				"output": cty.UnknownVal(cty.String),
			})),
		},
		&stackplan.PlannedChangeResourceInstancePlanned{
			ComponentInstanceAddr: stackaddrs.Absolute(
				stackaddrs.RootStackInstance,
				stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{Name: "self"},
				},
			),
			ChangeSrc: &plans.ResourceInstanceChangeSrc{
				Addr: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "terraform_data",
					Name: "main",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				PrevRunAddr: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "terraform_data",
					Name: "main",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				ProviderAddr: addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: addrs.NewBuiltInProvider("terraform"),
				},
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Create,
					Before: mustPlanDynamicValue(cty.NullVal(cty.DynamicPseudoType)),
					After: plans.DynamicValue{
						// This is an object conforming to the terraform_data
						// resource type's schema.
						//
						// FIXME: Should write this a different way that is
						// scrutable and won't break each time something gets
						// added to the terraform_data schema. (We can't use
						// mustPlanDynamicValue here because the resource type
						// uses DynamicPseudoType attributes, which require
						// explicitly-typed encoding.)
						0x84, 0xa2, 0x69, 0x64, 0xc7, 0x03, 0x0c, 0x81,
						0x01, 0xc2, 0xa5, 0x69, 0x6e, 0x70, 0x75, 0x74,
						0x92, 0xc4, 0x08, 0x22, 0x73, 0x74, 0x72, 0x69,
						0x6e, 0x67, 0x22, 0xa5, 0x68, 0x65, 0x6c, 0x6c,
						0x6f, 0xa6, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74,
						0x92, 0xc4, 0x08, 0x22, 0x73, 0x74, 0x72, 0x69,
						0x6e, 0x67, 0x22, 0xd4, 0x00, 0x00, 0xb0, 0x74,
						0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x73, 0x5f,
						0x72, 0x65, 0x70, 0x6c, 0x61, 0x63, 0x65, 0xc0,
					},
				},
			},
		},
	}

	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestPlanVariableOutputRoundtripNested(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "variable-output-roundtrip-nested")

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	req := PlanRequest{
		Config: cfg,
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &req, &resp)
	gotChanges, diags := collectPlanOutput(changesCh, diagsCh)

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics\n%s", diags.ErrWithWarnings().Error())
	}

	wantChanges := []stackplan.PlannedChange{
		&stackplan.PlannedChangeHeader{
			TerraformVersion: version.SemVer,
		},
		&stackplan.PlannedChangeOutputValue{
			Addr:     stackaddrs.OutputValue{Name: "msg"},
			Action:   plans.Create,
			OldValue: plans.DynamicValue{0xc0},                  // MessagePack nil
			NewValue: plans.DynamicValue([]byte("\xa7default")), // MessagePack string "default"
		},
		&stackplan.PlannedChangeApplyable{
			Applyable: true,
		},
	}

	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestPlanWithProviderConfig(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "with-provider-config")
	providerAddr := addrs.MustParseProviderSourceString("example.com/test/test")
	providerSchema := &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"name": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
		},
	}
	inputVarAddr := stackaddrs.InputVariable{Name: "name"}
	fakeSrcRng := tfdiags.SourceRange{
		Filename: "fake-source",
	}

	t.Run("valid", func(t *testing.T) {
		changesCh := make(chan stackplan.PlannedChange, 8)
		diagsCh := make(chan tfdiags.Diagnostic, 2)

		// FIXME: The MockProvider type is still lurking in
		// the terraform package; it would make more sense for
		// it to be providers.Mock, in the providers package.
		provider := &terraform.MockProvider{
			GetProviderSchemaResponse:      providerSchema,
			ValidateProviderConfigResponse: &providers.ValidateProviderConfigResponse{},
			ConfigureProviderResponse:      &providers.ConfigureProviderResponse{},
		}

		req := PlanRequest{
			Config: cfg,
			InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
				inputVarAddr: {
					Value:    cty.StringVal("Jackson"),
					DefRange: fakeSrcRng,
				},
			},
			ProviderFactories: map[addrs.Provider]providers.Factory{
				providerAddr: func() (providers.Interface, error) {
					return provider, nil
				},
			},
		}
		resp := PlanResponse{
			PlannedChanges: changesCh,
			Diagnostics:    diagsCh,
		}
		go Plan(ctx, &req, &resp)
		_, diags := collectPlanOutput(changesCh, diagsCh)
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics\n%s", diags.ErrWithWarnings().Error())
		}

		if !provider.ValidateProviderConfigCalled {
			t.Error("ValidateProviderConfig wasn't called")
		} else {
			req := provider.ValidateProviderConfigRequest
			if got, want := req.Config.GetAttr("name"), cty.StringVal("Jackson"); !got.RawEquals(want) {
				t.Errorf("wrong name in ValidateProviderConfig\ngot:  %#v\nwant: %#v", got, want)
			}
		}
		if !provider.ConfigureProviderCalled {
			t.Error("ConfigureProvider wasn't called")
		} else {
			req := provider.ConfigureProviderRequest
			if got, want := req.Config.GetAttr("name"), cty.StringVal("Jackson"); !got.RawEquals(want) {
				t.Errorf("wrong name in ConfigureProvider\ngot:  %#v\nwant: %#v", got, want)
			}
		}
		if !provider.CloseCalled {
			t.Error("provider wasn't closed")
		}
	})
}

// collectPlanOutput consumes the two output channels emitting results from
// a call to [Plan], and collects all of the data written to them before
// returning once changesCh has been closed by the sender to indicate that
// the planning process is complete.
func collectPlanOutput(changesCh <-chan stackplan.PlannedChange, diagsCh <-chan tfdiags.Diagnostic) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var changes []stackplan.PlannedChange
	var diags tfdiags.Diagnostics

	for {
		select {
		case change, ok := <-changesCh:
			if !ok {
				// The plan operation is complete but we might still have
				// some buffered diagnostics to consume.
				if diagsCh != nil {
					for diag := range diagsCh {
						diags = append(diags, diag)
					}
				}
				return changes, diags
			}
			changes = append(changes, change)
		case diag, ok := <-diagsCh:
			if !ok {
				// no more diagnostics to read
				diagsCh = nil
				continue
			}
			diags = append(diags, diag)
		}
	}
}
