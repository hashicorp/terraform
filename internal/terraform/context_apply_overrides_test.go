// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
)

// This file contains 'integration' tests for the Terraform test overrides
// functionality.
//
// These tests could live in context_apply_test or context_apply2_test but given
// the size of those files, it makes sense to keep these tests grouped together.

func TestContextOverrides(t *testing.T) {

	// The approach to the testing here, is to create some configuration that
	// would panic if executed normally because of the underlying provider.
	//
	// We then write overrides that make sure the underlying provider is never
	// called.
	//
	// We then run a plan, apply, refresh, destroy sequence that tests all the
	// potential function calls to the underlying provider to make sure we
	// have covered everything.
	//
	// Finally, we validate some expected values after the apply stage to make
	// sure the overrides are returning the values we want them to.

	tcs := map[string]struct {
		configs     map[string]string
		overrides   *mocking.Overrides
		outputs     cty.Value
		expectedErr string
	}{
		"resource": {
			configs: map[string]string{
				"main.tf": `
resource "test_instance" "instance" {
	value = "Hello, world!"
}

output "value" {
	value = test_instance.instance.value
}

output "id" {
	value = test_instance.instance.id
}`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustResourceInstanceAddr("test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("h3ll0"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"resource_from_provider": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

resource "test_instance" "instance" {
	value = "Hello, world!"
}

output "value" {
	value = test_instance.instance.value
}

output "id" {
	value = test_instance.instance.id
}`,
			},
			overrides: mocking.OverridesForTesting(func(overrides map[string]addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides["test"] = addrs.MakeMap[addrs.Targetable, *configs.Override]()
				overrides["test"].Put(mustResourceInstanceAddr("test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}, nil),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("h3ll0"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"selectively_applies_provider": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

provider "test" {
  alias = "secondary"
}

resource "test_instance" "primary" {
	value = "primary"
}

resource "test_instance" "secondary" {
    provider = test.secondary
	value = "secondary"
}

output "primary_value" {
	value = test_instance.primary.value
}

output "primary_id" {
	value = test_instance.primary.id
}

output "secondary_value" {
	value = test_instance.secondary.value
}

output "secondary_id" {
	value = test_instance.secondary.id
}`,
			},
			overrides: mocking.OverridesForTesting(func(overrides map[string]addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides["test.secondary"] = addrs.MakeMap[addrs.Targetable, *configs.Override]()
				// Test should not apply this override, as this provider is
				// not being used for this resource.
				overrides["test.secondary"].Put(mustResourceInstanceAddr("test_instance.primary"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("primary_id"),
					}),
				})
				overrides["test.secondary"].Put(mustResourceInstanceAddr("test_instance.secondary"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("secondary_id"),
					}),
				})
			}, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustResourceInstanceAddr("test_instance.primary"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"primary_id":      cty.StringVal("h3ll0"),
				"primary_value":   cty.StringVal("primary"),
				"secondary_id":    cty.StringVal("secondary_id"),
				"secondary_value": cty.StringVal("secondary"),
			}),
		},
		"propagates_provider_to_modules_explicit": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

module "mod" {
  source = "./mod"

  providers = {
    test = test
  }
}

output "value" {
	value = module.mod.value
}

output "id" {
	value = module.mod.id
}`,
				"mod/main.tf": `
provider "test" {}

resource "test_instance" "instance" {
	value = "Hello, world!"
}

output "value" {
	value = test_instance.instance.value
}

output "id" {
	value = test_instance.instance.id
}
`,
			},
			overrides: mocking.OverridesForTesting(func(overrides map[string]addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides["test"] = addrs.MakeMap[addrs.Targetable, *configs.Override]()
				overrides["test"].Put(mustResourceInstanceAddr("module.mod.test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}, nil),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("h3ll0"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"propagates_provider_to_modules_implicit": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

module "mod" {
  source = "./mod"
}

output "value" {
	value = module.mod.value
}

output "id" {
	value = module.mod.id
}`,
				"mod/main.tf": `
resource "test_instance" "instance" {
	value = "Hello, world!"
}

output "value" {
	value = test_instance.instance.value
}

output "id" {
	value = test_instance.instance.id
}

`,
			},
			overrides: mocking.OverridesForTesting(func(overrides map[string]addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides["test"] = addrs.MakeMap[addrs.Targetable, *configs.Override]()
				overrides["test"].Put(mustResourceInstanceAddr("module.mod.test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}, nil),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("h3ll0"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"data_source": {
			configs: map[string]string{
				"main.tf": `
data "test_instance" "instance" {
	id = "data-source"
}

resource "test_instance" "instance" {
	value = data.test_instance.instance.value
}

output "value" {
	value = test_instance.instance.value
}

output "id" {
	value = test_instance.instance.id
}`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustResourceInstanceAddr("test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
				overrides.Put(mustResourceInstanceAddr("data.test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"value": cty.StringVal("Hello, world!"),
					}),
				})
			}),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("h3ll0"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"module": {
			configs: map[string]string{
				"main.tf": `
module "mod" {
  source = "./mod"
}

output "value" {
	value = module.mod.value
}

output "id" {
	value = module.mod.id
}`,
				"mod/main.tf": `
resource "test_instance" "instance" {
	value = "random"
}

output "value" {
	value = test_instance.instance.value
}

output "id" {
	value = test_instance.instance.id
}

check "value" {
  assert {
    condition = test_instance.instance.value == "definitely wrong"
    error_message = "bad value"
  }
}
`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance("module.mod"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("h3ll0"),
						"value": cty.StringVal("Hello, world!"),
					}),
				})
			}),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("h3ll0"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"provider_type_override": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

module "mod" {
  source = "./mod"
}

output "value" {
	value = module.mod.value
}

output "id" {
	value = module.mod.id
}`,
				"mod/main.tf": `
terraform {
  required_providers {
    replaced = {
      source = "hashicorp/test"
    }
  }
}

resource "test_instance" "instance" {
    provider = replaced
	value = "Hello, world!"
}

output "value" {
	value = test_instance.instance.value
}

output "id" {
	value = test_instance.instance.id
}

`,
			},
			overrides: mocking.OverridesForTesting(func(overrides map[string]addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides["test"] = addrs.MakeMap[addrs.Targetable, *configs.Override]()
				overrides["test"].Put(mustResourceInstanceAddr("module.mod.test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}, nil),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("h3ll0"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"resource_instance_overrides": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

resource "test_instance" "instance" {
    count = 3
	value = "Hello, world!"
}

output "value" {
	value = test_instance.instance.*.value
}

output "id" {
	value = test_instance.instance.*.id
}`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustAbsResourceAddr("test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("generic"),
					}),
				})
				overrides.Put(mustResourceInstanceAddr("test_instance.instance[1]"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("specific"),
					}),
				})
			}),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id": cty.TupleVal([]cty.Value{
					cty.StringVal("generic"),
					cty.StringVal("specific"),
					cty.StringVal("generic"),
				}),
				"value": cty.TupleVal([]cty.Value{
					cty.StringVal("Hello, world!"),
					cty.StringVal("Hello, world!"),
					cty.StringVal("Hello, world!"),
				}),
			}),
		},
		"imports": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

import {
  id = "29C1E645FF91"
  to = test_instance.instance
}

resource "test_instance" "instance" {
  value = "Hello, world!"
}

output "id" {
  value = test_instance.instance.id
}
`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustAbsResourceAddr("test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("29C1E645FF91"),
					}),
				})
			}),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("29C1E645FF91"),
			}),
		},
		// This test is designed to fail as documentation that we do not support
		// config generation during tests. It's actually impossible in normal
		// usage to do this since `terraform test` never triggers config
		// generation.
		"imports_config_gen": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

import {
  id = "29C1E645FF91"
  to = test_instance.instance
}

output "id" {
  value = test_instance.instance.id
}
`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustAbsResourceAddr("test_instance.instance"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("29C1E645FF91"),
					}),
				})
			}),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("29C1E645FF91"),
			}),
			expectedErr: "override blocks do not support config generation",
		},
		"module_instance_overrides": {
			configs: map[string]string{
				"main.tf": `
provider "test" {}

module "mod" {
  count = 3
  source = "./mod"
}

output "value" {
	value = module.mod.*.value
}

output "id" {
	value = module.mod.*.id
}`,
				"mod/main.tf": `
terraform {
  required_providers {
    replaced = {
      source = "hashicorp/test"
    }
  }
}

resource "test_instance" "instance" {
    provider = replaced
	value = "Hello, world!"
}

output "value" {
	value = test_instance.instance.value
}

output "id" {
	value = test_instance.instance.id
}

`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance("module.mod"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("generic"),
						"value": cty.StringVal("Hello, world!"),
					}),
				})
				overrides.Put(mustModuleInstance("module.mod[1]"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("specific"),
						"value": cty.StringVal("Hello, world!"),
					}),
				})
			}),
			outputs: cty.ObjectVal(map[string]cty.Value{
				"id": cty.TupleVal([]cty.Value{
					cty.StringVal("generic"),
					cty.StringVal("specific"),
					cty.StringVal("generic"),
				}),
				"value": cty.TupleVal([]cty.Value{
					cty.StringVal("Hello, world!"),
					cty.StringVal("Hello, world!"),
					cty.StringVal("Hello, world!"),
				}),
			}),
		},
		"expansion inside overridden module": {
			configs: map[string]string{
				"main.tf": `
module "test" {
  source = "./mod"
}
`,
				"mod/main.tf": `
locals {
  instances = 2
  value = "Hello, world!"
}

resource "test_instance" "resource" {
  count = local.instances
  string = local.value
}

output "id" {
  value = test_instance.resource[0].id
}
`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance("module.test"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}),
			outputs: cty.EmptyObjectVal,
		},
		"expansion inside deeply nested overridden module": {
			configs: map[string]string{
				"main.tf": `
module "test" {
  source = "./child"
}
`,
				"child/main.tf": `
module "grandchild" {
  source = "../grandchild"
}

locals {
  instances = 2
  value = "Hello, world!"
}

resource "test_instance" "resource" {
  count = local.instances
  string = local.value
}

output "id" {
  value = test_instance.resource[0].id
}
`,
				"grandchild/main.tf": `
locals {
  instances = 2
  value = "Hello, world!"
}

resource "test_instance" "resource" {
  count = local.instances
  string = local.value
}

output "id" {
  value = test_instance.resource[0].id
}
`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance("module.test"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}),
			outputs: cty.EmptyObjectVal,
		},
		"legacy provider config inside overridden module": {
			configs: map[string]string{
				"main.tf": `
module "test" {
  source = "./child"
}
`,
				"child/main.tf": `
module "grandchild" {
  source = "../grandchild"
}
output "id" {
  value = "child"
}
`,
				"grandchild/main.tf": `
variable "in" {
  default = "test_value"
}

provider "test" {
  value = var.in
}

resource "test_instance" "resource" {
}
`,
			},
			overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance("module.test"), &configs.Override{
					Values: cty.ObjectVal(map[string]cty.Value{
						"id": cty.StringVal("h3ll0"),
					}),
				})
			}),
			outputs: cty.EmptyObjectVal,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			cfg := testModuleInline(t, tc.configs)
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(underlyingOverridesProvider),
				},
			})

			plan, diags := ctx.Plan(cfg, states.NewState(), &PlanOpts{
				Mode:               plans.NormalMode,
				Overrides:          tc.overrides,
				GenerateConfigPath: "out.tf",
			})
			if len(tc.expectedErr) > 0 {
				if diags.ErrWithWarnings().Error() != tc.expectedErr {
					t.Fatal(diags)
				}
				return // Don't do the rest of the test if we were expecting errors.
			}

			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			state, diags := ctx.Apply(plan, cfg, nil)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			outputs := make(map[string]cty.Value, len(cfg.Module.Outputs))
			for _, output := range cfg.Module.Outputs {
				outputs[output.Name] = state.OutputValue(output.Addr().Absolute(addrs.RootModuleInstance)).Value
			}
			actual := cty.ObjectVal(outputs)

			if !actual.RawEquals(tc.outputs) {
				t.Fatalf("expected:\n%s\nactual:\n%s", tc.outputs.GoString(), actual.GoString())
			}

			_, diags = ctx.Plan(cfg, state, &PlanOpts{
				Mode:      plans.RefreshOnlyMode,
				Overrides: tc.overrides,
			})
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			destroyPlan, diags := ctx.Plan(cfg, state, &PlanOpts{
				Mode:      plans.DestroyMode,
				Overrides: tc.overrides,
			})
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			_, diags = ctx.Apply(destroyPlan, cfg, nil)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
		})
	}

}

// underlyingOverridesProvider returns a provider that always panics for
// important calls. This is to validate the behaviour of the overrides
// functionality, in that they should stop the provider from being executed.
var underlyingOverridesProvider = &testing_provider.MockProvider{
	GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"value": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
						"value": {
							Type:     cty.String,
							Computed: true,
						},
					},
				},
			},
		},
	},
	ReadResourceFn: func(request providers.ReadResourceRequest) providers.ReadResourceResponse {
		panic("ReadResourceFn called, should have been overridden.")
	},
	PlanResourceChangeFn: func(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		panic("PlanResourceChangeFn called, should have been overridden.")
	},
	ApplyResourceChangeFn: func(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		panic("ApplyResourceChangeFn called, should have been overridden.")
	},
	ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		panic("ReadDataSourceFn called, should have been overridden.")
	},
	ImportResourceStateFn: func(request providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		panic("ImportResourceStateFn called, should have been overridden.")
	},
}
