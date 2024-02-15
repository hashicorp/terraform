// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	terraformProvider "github.com/hashicorp/terraform/internal/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

func TestPlanWithMissingInputVariable(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "plan-undeclared-variable-in-component")

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1994-09-05T08:50:00Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("terraform"): func() (providers.Interface, error) {
				return terraformProvider.NewProvider(), nil
			},
		},

		ForcePlanTimestamp: &fakePlanTimestamp,
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &req, &resp)
	_, gotDiags := collectPlanOutput(changesCh, diagsCh)

	// We'll normalize the diagnostics to be of consistent underlying type
	// using ForRPC, so that we can easily diff them; we don't actually care
	// about which underlying implementation is in use.
	gotDiags = gotDiags.ForRPC()
	var wantDiags tfdiags.Diagnostics
	wantDiags = wantDiags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Reference to undeclared input variable",
		Detail:   `There is no variable "input" block declared in this stack.`,
		Subject: &hcl.Range{
			Filename: mainBundleSourceAddrStr("plan-undeclared-variable-in-component/undeclared-variable.tfstack.hcl"),
			Start:    hcl.Pos{Line: 17, Column: 13, Byte: 250},
			End:      hcl.Pos{Line: 17, Column: 22, Byte: 259},
		},
	})
	wantDiags = wantDiags.ForRPC()

	if diff := cmp.Diff(wantDiags, gotDiags); diff != "" {
		t.Errorf("wrong diagnostics\n%s", diff)
	}
}

func TestPlanWithSingleResource(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "with-single-resource")

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1994-09-05T08:50:00Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("terraform"): func() (providers.Interface, error) {
				return terraformProvider.NewProvider(), nil
			},
		},

		ForcePlanTimestamp: &fakePlanTimestamp,
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
			Action:             plans.Create,
			PlanApplyable:      true,
			PlanComplete:       true,
			PlannedInputValues: make(map[string]plans.DynamicValue),
			PlannedOutputValues: map[string]cty.Value{
				"input":  cty.StringVal("hello"),
				"output": cty.UnknownVal(cty.String),
			},
			PlanTimestamp: fakePlanTimestamp,
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
			ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
				Component: stackaddrs.Absolute(
					stackaddrs.RootStackInstance,
					stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{Name: "self"},
					},
				),
				Item: addrs.AbsResourceInstanceObject{
					ResourceInstance: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "terraform_data",
						Name: "main",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				},
			},
			ProviderConfigAddr: addrs.AbsProviderConfig{
				Module:   addrs.RootModule,
				Provider: addrs.MustParseProviderSourceString("terraform.io/builtin/terraform"),
			},
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

			// The following is schema for the real terraform_data resource
			// type from the real terraform.io/builtin/terraform provider
			// maintained elsewhere in this codebase. If that schema changes
			// in future then this should change to match it.
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"input":            {Type: cty.DynamicPseudoType, Optional: true},
					"output":           {Type: cty.DynamicPseudoType, Computed: true},
					"triggers_replace": {Type: cty.DynamicPseudoType, Optional: true},
					"id":               {Type: cty.String, Computed: true},
				},
			},
		},
	}

	cmpOptions := cmp.Options{
		ctydebug.CmpOptions,
		collections.CmpOptions,
	}
	if diff := cmp.Diff(wantChanges, gotChanges, cmpOptions); diff != "" {
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
		&stackplan.PlannedChangeApplyable{
			Applyable: true,
		},
		&stackplan.PlannedChangeHeader{
			TerraformVersion: version.SemVer,
		},
		&stackplan.PlannedChangeOutputValue{
			Addr:     stackaddrs.OutputValue{Name: "msg"},
			Action:   plans.Create,
			OldValue: plans.DynamicValue{0xc0},                  // MessagePack nil
			NewValue: plans.DynamicValue([]byte("\xa7default")), // MessagePack string "default"
		},
		&stackplan.PlannedChangeRootInputValue{
			Addr: stackaddrs.InputVariable{
				Name: "msg",
			},
			Value: cty.NullVal(cty.String),
		},
	}
	sort.SliceStable(gotChanges, func(i, j int) bool {
		// An arbitrary sort just to make the result stable for comparison.
		return fmt.Sprintf("%T", gotChanges[i]) < fmt.Sprintf("%T", gotChanges[j])
	})

	if diff := cmp.Diff(wantChanges, gotChanges, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

var cmpCollectionsSet = cmp.Comparer(func(x, y collections.Set[stackaddrs.AbsComponent]) bool {
	if x.Len() != y.Len() {
		return false
	}

	for _, v := range x.Elems() {
		if !y.Has(v) {
			return false
		}
	}

	return true
})

func TestPlanSensitiveOutput(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "sensitive-output")

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	req := PlanRequest{
		Config:             cfg,
		ForcePlanTimestamp: &fakePlanTimestamp,
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
			Action:             plans.Create,
			PlanApplyable:      true,
			PlanComplete:       true,
			PlannedInputValues: make(map[string]plans.DynamicValue),
			PlannedOutputValues: map[string]cty.Value{
				"out": cty.StringVal("secret").Mark(marks.Sensitive),
			},
			PlanTimestamp: fakePlanTimestamp,
		},
		&stackplan.PlannedChangeHeader{
			TerraformVersion: version.SemVer,
		},
		&stackplan.PlannedChangeOutputValue{
			Addr:          stackaddrs.OutputValue{Name: "result"},
			Action:        plans.Create,
			OldValue:      plans.DynamicValue{0xc0}, // MessagePack nil
			NewValue:      mustPlanDynamicValue(cty.StringVal("secret")),
			NewValueMarks: []cty.PathValueMarks{{Marks: cty.NewValueMarks(marks.Sensitive)}},
		},
	}
	sort.SliceStable(gotChanges, func(i, j int) bool {
		// An arbitrary sort just to make the result stable for comparison.
		return fmt.Sprintf("%T", gotChanges[i]) < fmt.Sprintf("%T", gotChanges[j])
	})

	if diff := cmp.Diff(wantChanges, gotChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestPlanSensitiveOutputNested(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "sensitive-output-nested")

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	req := PlanRequest{
		Config:             cfg,
		ForcePlanTimestamp: &fakePlanTimestamp,
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
		&stackplan.PlannedChangeApplyable{
			Applyable: true,
		},
		&stackplan.PlannedChangeComponentInstance{
			Addr: stackaddrs.Absolute(
				stackaddrs.RootStackInstance.Child("child", addrs.NoKey),
				stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{Name: "self"},
				},
			),
			Action:             plans.Create,
			PlanApplyable:      true,
			PlanComplete:       true,
			PlannedInputValues: make(map[string]plans.DynamicValue),
			PlannedOutputValues: map[string]cty.Value{
				"out": cty.StringVal("secret").Mark(marks.Sensitive),
			},
			PlanTimestamp: fakePlanTimestamp,
		},
		&stackplan.PlannedChangeHeader{
			TerraformVersion: version.SemVer,
		},
		&stackplan.PlannedChangeOutputValue{
			Addr:          stackaddrs.OutputValue{Name: "result"},
			Action:        plans.Create,
			OldValue:      plans.DynamicValue{0xc0}, // MessagePack nil
			NewValue:      mustPlanDynamicValue(cty.StringVal("secret")),
			NewValueMarks: []cty.PathValueMarks{{Marks: cty.NewValueMarks(marks.Sensitive)}},
		},
	}
	sort.SliceStable(gotChanges, func(i, j int) bool {
		// An arbitrary sort just to make the result stable for comparison.
		return fmt.Sprintf("%T", gotChanges[i]) < fmt.Sprintf("%T", gotChanges[j])
	})

	if diff := cmp.Diff(wantChanges, gotChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestPlanSensitiveOutputAsInput(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "sensitive-output-as-input")

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange, 8)
	diagsCh := make(chan tfdiags.Diagnostic, 2)
	req := PlanRequest{
		Config:             cfg,
		ForcePlanTimestamp: &fakePlanTimestamp,
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
		&stackplan.PlannedChangeApplyable{
			Applyable: true,
		},
		&stackplan.PlannedChangeComponentInstance{
			Addr: stackaddrs.Absolute(
				stackaddrs.RootStackInstance.Child("sensitive", addrs.NoKey),
				stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{Name: "self"},
				},
			),
			Action:             plans.Create,
			PlanApplyable:      true,
			PlanComplete:       true,
			PlannedInputValues: make(map[string]plans.DynamicValue),
			PlannedOutputValues: map[string]cty.Value{
				"out": cty.StringVal("secret").Mark(marks.Sensitive),
			},
			PlanTimestamp: fakePlanTimestamp,
		},
		&stackplan.PlannedChangeComponentInstance{
			Addr: stackaddrs.Absolute(
				stackaddrs.RootStackInstance,
				stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{Name: "self"},
				},
			),
			Action:        plans.Create,
			PlanApplyable: true,
			PlanComplete:  true,
			PlannedInputValues: map[string]plans.DynamicValue{
				"secret": mustPlanDynamicValueDynamicType(cty.StringVal("secret")),
			},
			PlannedInputValueMarks: map[string][]cty.PathValueMarks{
				"secret": {
					{
						Marks: cty.NewValueMarks(marks.Sensitive),
					},
				},
			},
			PlannedOutputValues: map[string]cty.Value{
				"result": cty.StringVal("SECRET").Mark(marks.Sensitive),
			},
			PlanTimestamp: fakePlanTimestamp,
		},
		&stackplan.PlannedChangeHeader{
			TerraformVersion: version.SemVer,
		},
		&stackplan.PlannedChangeOutputValue{
			Addr:          stackaddrs.OutputValue{Name: "result"},
			Action:        plans.Create,
			OldValue:      plans.DynamicValue{0xc0}, // MessagePack nil
			NewValue:      mustPlanDynamicValue(cty.StringVal("SECRET")),
			NewValueMarks: []cty.PathValueMarks{{Marks: cty.NewValueMarks(marks.Sensitive)}},
		},
	}
	sort.SliceStable(gotChanges, func(i, j int) bool {
		// An arbitrary sort just to make the result stable for comparison.
		return fmt.Sprintf("%T", gotChanges[i]) < fmt.Sprintf("%T", gotChanges[j])
	})

	if diff := cmp.Diff(wantChanges, gotChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
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
func TestPlanWithRemovedResource(t *testing.T) {
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1994-09-05T08:50:00Z")
	if err != nil {
		t.Fatal(err)
	}

	attrs := map[string]interface{}{
		"id": "FE1D5830765C",
		"input": map[string]interface{}{
			"value": "hello",
			"type":  "string",
		},
		"output": map[string]interface{}{
			"value": nil,
			"type":  "string",
		},
		"triggers_replace": nil,
	}
	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		t.Fatal(err)
	}

	// We want to see that it's adding the extra context for when a provider is
	// missing for a resource that's in state and not in config.
	expectedDiagnostic := "has resources in state that"

	tcs := make(map[string]*string)
	tcs["missing-providers"] = &expectedDiagnostic
	tcs["valid-providers"] = nil

	for name, diag := range tcs {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cfg := loadMainBundleConfigForTest(t, path.Join("empty-component", name))

			req := PlanRequest{
				Config: cfg,
				ProviderFactories: map[addrs.Provider]providers.Factory{
					addrs.NewBuiltInProvider("terraform"): func() (providers.Interface, error) {
						return terraformProvider.NewProvider(), nil
					},
				},

				ForcePlanTimestamp: &fakePlanTimestamp,

				// PrevState specifies a state with a resource that is not present in
				// the current configuration. This is a common situation when a resource
				// is removed from the configuration but still exists in the state.
				PrevState: stackstate.NewStateBuilder().
					AddResourceInstance(stackstate.NewResourceInstanceBuilder().
						SetAddr(stackaddrs.AbsResourceInstanceObject{
							Component: stackaddrs.AbsComponentInstance{
								Stack: stackaddrs.RootStackInstance,
								Item: stackaddrs.ComponentInstance{
									Component: stackaddrs.Component{
										Name: "self",
									},
									Key: addrs.NoKey,
								},
							},
							Item: addrs.AbsResourceInstanceObject{
								ResourceInstance: addrs.AbsResourceInstance{
									Module: addrs.RootModuleInstance,
									Resource: addrs.ResourceInstance{
										Resource: addrs.Resource{
											Mode: addrs.ManagedResourceMode,
											Type: "terraform_data",
											Name: "main",
										},
										Key: addrs.NoKey,
									},
								},
								DeposedKey: addrs.NotDeposed,
							},
						}).
						SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
							SchemaVersion: 0,
							AttrsJSON:     attrsJSON,
							Status:        states.ObjectReady,
						}).
						SetProviderAddr(addrs.AbsProviderConfig{
							Module:   addrs.RootModule,
							Provider: addrs.MustParseProviderSourceString("terraform.io/builtin/terraform"),
						})).
					Build(),
			}

			changesCh := make(chan stackplan.PlannedChange)
			diagsCh := make(chan tfdiags.Diagnostic)
			resp := PlanResponse{
				PlannedChanges: changesCh,
				Diagnostics:    diagsCh,
			}

			go Plan(ctx, &req, &resp)
			_, diags := collectPlanOutput(changesCh, diagsCh)

			if diag != nil {
				if len(diags) == 0 {
					t.Fatalf("expected diagnostics, got none")
				}
				if !strings.Contains(diags[0].Description().Detail, *diag) {
					t.Fatalf("expected diagnostic %q, got %q", *diag, diags[0].Description().Detail)
				}
			} else if len(diags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", diags.ErrWithWarnings().Error())
			}
		})
	}
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
