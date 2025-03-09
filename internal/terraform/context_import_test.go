// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func TestContextImport_basic(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportStr)
	if actual != expected {
		t.Fatalf("wrong final state\ngot:\n%s\nwant:\n%s", actual, expected)
	}
}

// import 1 of count instances in the configuration
func TestContextImport_countIndex(t *testing.T) {
	p := testProvider("aws")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
provider "aws" {
  foo = "bar"
}

resource "aws_instance" "foo" {
  count = 2
}
`})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.IntKey(0),
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportCountIndexStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_collision(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsFlat: map[string]string{
					"id": "bar",
				},
				Status: states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("aws"),
				Module:   addrs.RootModule,
			},
		)
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	state, diags := ctx.Import(m, state, &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want an error indicating that the resource already exists in state")
	}

	actual := strings.TrimSpace(state.String())
	expected := `aws_instance.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]`

	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_missingType(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := "<no state>"
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleProvider(t *testing.T) {
	p := testProvider("aws")

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		foo := req.Config.GetAttr("foo").AsString()
		if foo != "bar" {
			resp.Diagnostics = resp.Diagnostics.Append(errors.New("not bar"))
		}

		return
	}

	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !p.ConfigureProviderCalled {
		t.Fatal("didn't configure provider")
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

// Importing into a module requires a provider config in that module.
func TestContextImport_providerModule(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-module")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		foo := req.Config.GetAttr("foo").AsString()
		if foo != "bar" {
			resp.Diagnostics = resp.Diagnostics.Append(errors.New("not bar"))
		}

		return
	}

	_, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.Child("child", addrs.IntKey(0)).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if !p.ConfigureProviderCalled {
		t.Fatal("didn't configure provider")
	}
}

// Test that import will interpolate provider configuration and use
// that configuration for import.
func TestContextImport_providerConfig(t *testing.T) {
	testCases := map[string]struct {
		module string
		value  string
	}{
		"variables": {
			module: "import-provider-vars",
			value:  "bar",
		},
		"locals": {
			module: "import-provider-locals",
			value:  "baz-bar",
		},
	}
	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			p := testProvider("aws")
			m := testModule(t, test.module)
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
				},
			})

			p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
				ImportedResources: []providers.ImportedResource{
					{
						TypeName: "aws_instance",
						State: cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("foo"),
						}),
					},
				},
			}

			state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
				Targets: []*ImportTarget{
					{
						LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
							addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
						),
						IDString: "bar",
					},
				},
				SetVariables: InputValues{
					"foo": &InputValue{
						Value:      cty.StringVal("bar"),
						SourceType: ValueFromCaller,
					},
				},
			})
			if diags.HasErrors() {
				t.Fatalf("unexpected errors: %s", diags.Err())
			}

			if !p.ConfigureProviderCalled {
				t.Fatal("didn't configure provider")
			}

			if foo := p.ConfigureProviderRequest.Config.GetAttr("foo").AsString(); foo != test.value {
				t.Fatalf("bad value %#v; want %#v", foo, test.value)
			}

			actual := strings.TrimSpace(state.String())
			expected := strings.TrimSpace(testImportStr)
			if actual != expected {
				t.Fatalf("bad: \n%s", actual)
			}
		})
	}
}

// Test that provider configs can't reference resources.
func TestContextImport_providerConfigResources(t *testing.T) {
	p := testProvider("aws")
	pTest := testProvider("test")
	m := testModule(t, "import-provider-resources")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"):  testProviderFuncFixed(p),
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(pTest),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	_, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("should error")
	}
	if got, want := diags.Err().Error(), `The configuration for provider["registry.terraform.io/hashicorp/aws"] depends on values that cannot be determined until apply.`; !strings.Contains(got, want) {
		t.Errorf("wrong error\n got: %s\nwant: %s", got, want)
	}
}

func TestContextImport_refresh(t *testing.T) {
	p := testProvider("aws")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
provider "aws" {
  foo = "bar"
}

resource "aws_instance" "foo" {
}


// we are only importing aws_instance.foo, so these resources will be unknown
resource "aws_instance" "bar" {
}
data "aws_data_source" "bar" {
  foo = aws_instance.bar.id
}
`})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	p.ReadDataSourceResponse = &providers.ReadDataSourceResponse{
		State: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("id"),
			"foo": cty.UnknownVal(cty.String),
		}),
	}

	p.ReadResourceFn = nil

	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.StringVal("foo"),
			"foo": cty.StringVal("bar"),
		}),
	}

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	if d := state.ResourceInstance(mustResourceInstanceAddr("data.aws_data_source.bar")); d != nil {
		t.Errorf("data.aws_data_source.bar has a status of ObjectPlanned and should not be in the state\ngot:%#v\n", d.Current)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportRefreshStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_refreshNil(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{
			NewState: cty.NullVal(cty.DynamicPseudoType),
		}
	}

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if !diags.HasErrors() {
		t.Fatal("should error")
	}

	actual := strings.TrimSpace(state.String())
	expected := "<no state>"
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_module(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-module")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.Child("child", addrs.IntKey(0)).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportModuleStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleDepth2(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-module")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.Child("child", addrs.IntKey(0)).Child("nested", addrs.NoKey).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "baz",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportModuleDepth2Str)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_moduleDiff(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-module")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.Child("child", addrs.IntKey(0)).ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "baz",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportModuleStr)
	if actual != expected {
		t.Fatalf("\nexpected: %q\ngot:      %q\n", expected, actual)
	}
}

func TestContextImport_multiState(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
			"aws_instance_thing": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
			{
				TypeName: "aws_instance_thing",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("bar"),
				}),
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportMultiStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_multiStateSame(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "import-provider")

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
			"aws_instance_thing": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
			{
				TypeName: "aws_instance_thing",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("bar"),
				}),
			},
			{
				TypeName: "aws_instance_thing",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("qux"),
				}),
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportMultiSameStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_nestedModuleImport(t *testing.T) {
	p := testProvider("aws")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  xs = toset(["foo"])
}

module "a" {
  for_each = local.xs
  source   = "./a"
}

module "b" {
  for_each = local.xs
  source   = "./b"
  y = module.a[each.key].y
}

resource "test_resource" "test" {
}
`,
		"a/main.tf": `
output "y" {
  value = "bar"
}
`,
		"b/main.tf": `
variable "y" {
  type = string
}

resource "test_resource" "unused" {
  value = var.y
  // missing required, but should not error
}
`,
	})

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id":       {Type: cty.String, Computed: true},
					"required": {Type: cty.String, Required: true},
				},
			},
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_resource",
				State: cty.ObjectVal(map[string]cty.Value{
					"id":       cty.StringVal("test"),
					"required": cty.StringVal("value"),
				}),
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "test_resource", "test", addrs.NoKey,
				),
				IDString: "test",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	ri := state.ResourceInstance(mustResourceInstanceAddr("test_resource.test"))
	expected := `{"id":"test","required":"value"}`
	if ri == nil || ri.Current == nil {
		t.Fatal("no state is recorded for resource instance test_resource.test")
	}
	if string(ri.Current.AttrsJSON) != expected {
		t.Fatalf("expected %q, got %q\n", expected, ri.Current.AttrsJSON)
	}
}

// New resources in the config during import won't exist for evaluation
// purposes (until import is upgraded to using a complete plan). This means
// that references to them are unknown, but in the case of single instances, we
// can at least know the type of unknown value.
func TestContextImport_newResourceUnknown(t *testing.T) {
	p := testProvider("aws")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_resource" "one" {
}

resource "test_resource" "two" {
  count = length(flatten([test_resource.one.id]))
}

resource "test_resource" "test" {
}
`})

	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_resource",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("test"),
				}),
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "test_resource", "test", addrs.NoKey,
				),
				IDString: "test",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	ri := state.ResourceInstance(mustResourceInstanceAddr("test_resource.test"))
	expected := `{"id":"test"}`
	if ri == nil || ri.Current == nil {
		t.Fatal("no state is recorded for resource instance test_resource.test")
	}
	if string(ri.Current.AttrsJSON) != expected {
		t.Fatalf("expected %q, got %q\n", expected, ri.Current.AttrsJSON)
	}
}

func TestContextImport_33572(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "issue-33572")

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "aws_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
		},
	}

	state, diags := ctx.Import(m, states.NewState(), &ImportOpts{
		Targets: []*ImportTarget{
			{
				LegacyAddr: addrs.RootModuleInstance.ResourceInstance(
					addrs.ManagedResourceMode, "aws_instance", "foo", addrs.NoKey,
				),
				IDString: "bar",
			},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Err())
	}
	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportStrWithDataSource)
	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Fatalf("wrong final state\ngot:\n%s\nwant:\n%s\ndiff:\n%s", actual, expected, diff)
	}
}

// Missing import target should produce an error
func TestContextImport_missingModuleImport(t *testing.T) {
	p := testProvider("test")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  xs = toset(["foo"])
}

module "a" {
  for_each = local.xs
  source   = "./a"
}

import {
	to = module.WRONG.test_resource.x
	id = "id"
}
`,
		"a/main.tf": `
resource "test_resource" "x" {
  value = "z"
}
`,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Plan(m, states.NewState(), DefaultPlanOpts)
	if !diags.HasErrors() {
		t.Fatal("expected missing import target error")
	}
	if !strings.Contains(diags.Err().Error(), "Configuration for import target does not exist") {
		t.Fatalf("incorrect error for missing import target: %s\n", diags.Err())
	}
}

const testImportStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
`

const testImportStrWithDataSource = `
data.aws_data_source.bar:
  ID = baz
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
`

const testImportCountIndexStr = `
aws_instance.foo.0:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
`

const testImportModuleStr = `
<no state>
module.child[0]:
  aws_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
`

const testImportModuleDepth2Str = `
<no state>
module.child[0].nested:
  aws_instance.foo:
    ID = foo
    provider = provider["registry.terraform.io/hashicorp/aws"]
`

const testImportMultiStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance_thing.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
`

const testImportMultiSameStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance_thing.foo:
  ID = bar
  provider = provider["registry.terraform.io/hashicorp/aws"]
aws_instance_thing.foo-1:
  ID = qux
  provider = provider["registry.terraform.io/hashicorp/aws"]
`

const testImportRefreshStr = `
aws_instance.foo:
  ID = foo
  provider = provider["registry.terraform.io/hashicorp/aws"]
  foo = bar
`
