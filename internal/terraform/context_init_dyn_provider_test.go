package terraform

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestInit_DynamicProviderSource(t *testing.T) {
	for name, tc := range map[string]struct {
		module         map[string]string
		vars           InputValues
		validationFunc func(
			t *testing.T,
			cfg *configs.Config,
			diags tfdiags.Diagnostics,
		)
	}{
		"resolve required provider source": {
			module: map[string]string{
				"main.tf": `
terraform {
	required_providers {
		test-provider = {
			source = var.test-provider_src	
		}
	}
}

variable "test-provider_src" {
	type = string
	const = true
}
`,
			},
			vars: InputValues{
				"test-provider_src": {
					Value:      cty.StringVal("test-provider"),
					SourceType: ValueFromCLIArg,
				},
			},
			validationFunc: func(
				t *testing.T,
				cfg *configs.Config,
				diags tfdiags.Diagnostics,
			) {
				if diags.HasErrors() {
					t.Fatal(diags)
				}

				rp := expectRequiredProviderInModule(t, "test-provider", cfg.Module)
				expectRequiredProviderSource(t, "test-provider", rp.Source)
				expectRequiredProviderVersion(t, "", rp.Requirement.Required)
			},
		},
		"resolve required provider source and version from variables": {
			module: map[string]string{
				"main.tf": `
terraform {
	required_providers {
		test-provider = {
			source = var.test-provider_src	
			version = var.test-provider_version
		}
	}
}

variable "test-provider_src" {
	type = string
	const = true
}

variable "test-provider_version" {
	type = string
	const = true
}
`,
			},
			vars: InputValues{
				"test-provider_src": {
					Value:      cty.StringVal("test-provider"),
					SourceType: ValueFromCLIArg,
				},
				"test-provider_version": {
					Value:      cty.StringVal("0.0.1"),
					SourceType: ValueFromCLIArg,
				},
			},
			validationFunc: func(
				t *testing.T,
				cfg *configs.Config,
				diags tfdiags.Diagnostics,
			) {
				if diags.HasErrors() {
					t.Fatal(diags)
				}

				rp := expectRequiredProviderInModule(t, "test-provider", cfg.Module)
				expectRequiredProviderSource(t, "test-provider", rp.Source)
				expectRequiredProviderVersion(t, "0.0.1", rp.Requirement.Required)
			},
		},
		"resolve required provider source including string interpolation": {
			module: map[string]string{
				"main.tf": `
terraform {
	required_providers {
		test-provider = {
			source = "${var.test-provider_src}/interpolation"
		}
	}
}

variable "test-provider_src" {
	type = string
	const = true
}
`,
			},
			vars: InputValues{
				"test-provider_src": {
					Value:      cty.StringVal("test"),
					SourceType: ValueFromCLIArg,
				},
			},
			validationFunc: func(
				t *testing.T,
				cfg *configs.Config,
				diags tfdiags.Diagnostics,
			) {
				if diags.HasErrors() {
					t.Fatal(diags.Err())
				}

				rp := expectRequiredProviderInModule(t, "test-provider", cfg.Module)
				expectRequiredProviderSource(t, "test/interpolation", rp.Source)
			},
		},
		"resolve required provider from local": {
			module: map[string]string{
				"main.tf": `
terraform {
	required_providers {
		test-provider = {
			source = local.provider_src
		}
	}
}

variable "test-provider_src" {
	type = string
	const = true
}

locals {
	provider_src = "${var.test-provider_src}/local"
}
`,
			},
			vars: InputValues{
				"test-provider_src": {
					Value:      cty.StringVal("test"),
					SourceType: ValueFromCLIArg,
				},
			},
			validationFunc: func(
				t *testing.T,
				cfg *configs.Config,
				diags tfdiags.Diagnostics,
			) {
				if diags.HasErrors() {
					t.Fatal(diags.Err())
				}

				rp := expectRequiredProviderInModule(t, "test-provider", cfg.Module)
				expectRequiredProviderSource(t, "test/local", rp.Source)
			},
		},
		"expect error when non const variable is being used": {
			module: map[string]string{
				"main.tf": `
terraform {
	required_providers {
		test-provider = {
			source = var.test-provider_src
		}
	}
}

variable "test-provider_src" {
	type = string
	default = "nonconst"
}
`,
			},
			vars: InputValues{
				"test-provider_src": {
					Value:      cty.StringVal("test"),
					SourceType: ValueFromCLIArg,
				},
			},
			validationFunc: func(
				t *testing.T,
				cfg *configs.Config,
				diags tfdiags.Diagnostics,
			) {
				if !diags.HasErrors() {
					t.Fatalf("expect error when non const variable is being used")
				}
				if diags.Err().Error() != "Invalid provider source: The provider source contains a reference that is unknown during init." {
					t.Fatalf(
						"expected error msg: %s, got %s",
						"Invalid provider source: The provider source contains a reference that is unknown during init.",
						diags.Err().Error(),
					)
				}
			},
		},
		"resolve required provider when static and dynamic providers are used": {
			module: map[string]string{
				"main.tf": `
terraform {
	required_providers {
		dyn-provider = {
			source = var.test-provider_src
			version = "~> 0.0.1-dynamic"
		}
		static-provider = {
			source = "test/static"
			version = "~> 0.0.1-static"
		}
	}
}

variable "test-provider_src" {
	type = string
	const = true
}
`,
			},
			vars: InputValues{
				"test-provider_src": {
					Value:      cty.StringVal("test/dynamic"),
					SourceType: ValueFromCLIArg,
				},
			},
			validationFunc: func(
				t *testing.T,
				cfg *configs.Config,
				diags tfdiags.Diagnostics,
			) {
				if diags.HasErrors() {
					t.Fatal(diags.Err())
				}

				rp := expectRequiredProviderInModule(t, "dyn-provider", cfg.Module)
				expectRequiredProviderSource(t, "test/dynamic", rp.Source)
				expectRequiredProviderVersion(t, "~> 0.0.1-dynamic", rp.Requirement.Required)

				rp = expectRequiredProviderInModule(t, "static-provider", cfg.Module)
				expectRequiredProviderSource(t, "test/static", rp.Source)
				expectRequiredProviderVersion(t, "~> 0.0.1-static", rp.Requirement.Required)
			},
		},
		"resolve required provider in the child module": {
			module: map[string]string{
				"main.tf": `
variable "test-provider_src" {
	type = string
	const = true
}

variable "test-provider_ver" {
	type = string
	const = true
}

module "child" {
	source = "./child"
	provider_src = var.test-provider_src
	provider_ver = var.test-provider_ver
}
`,
				"child/main.tf": `
terraform {
	required_providers {
		child-provider = {
			source = var.provider_src
			version = var.provider_ver
		}
	}
}

variable "provider_src" {
	type = string
	const = true
}

variable "provider_ver" {
	type = string
	const = true
}
`,
			},
			vars: InputValues{
				"test-provider_src": {
					Value:      cty.StringVal("test/child"),
					SourceType: ValueFromCLIArg,
				},
				"test-provider_ver": {
					Value:      cty.StringVal("0.4.2-child"),
					SourceType: ValueFromCLIArg,
				},
			},
			validationFunc: func(
				t *testing.T,
				cfg *configs.Config,
				diags tfdiags.Diagnostics,
			) {
				if diags.HasErrors() {
					t.Fatal(diags.Err())
				}

				childCfg, ok := cfg.Children["child"]
				if !ok {
					t.Fatalf("expected child module 'child' in config, not found")
				}

				rp, ok := childCfg.Module.ProviderRequirements.RequiredProviders["child-provider"]
				if !ok {
					t.Fatal("expected provider 'child-provider' in child module requirements, not found")
				}
				expectRequiredProviderSource(t, "test/child", rp.Source)
				expectRequiredProviderVersion(t, "0.4.2-child", rp.Requirement.Required)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m, d := testModuleInlineWithVarsReturnDiags(t, tc.module, tc.vars)
			if d != nil {
				tc.validationFunc(t, nil, d)
				return
			}

			ctx := testContext2(t, &ContextOpts{Parallelism: 1})
			walker := MockModuleWalker{
				DefaultModule: testRootModuleInline(
					t,
					map[string]string{"main.tf": `// empty`},
				),
			}

			// Mock root module calls to children if present
			if len(m.Children) > 0 {
				for cn, cc := range m.Root.Children {
					if child, ok := m.Children[cn]; ok {
						walker.MockModuleCalls(t, map[string]*configs.Module{
							child.SourceAddrRaw: cc.Module,
						})
					}
				}
			}

			cfg, diags := ctx.Init(m.Root.Module, InitOpts{
				SetVariables: tc.vars,
				Walker:       &walker,
			})

			tc.validationFunc(t, cfg, diags)
		})
	}
}

func expectRequiredProviderInModule(
	t *testing.T,
	expect string,
	module *configs.Module,
) *configs.RequiredProvider {
	if module.ProviderRequirements == nil {
		t.Fatal("no provider requirements were set")
	}

	rp, ok := module.ProviderRequirements.RequiredProviders[expect]
	if !ok {
		t.Fatalf("required provider %q not found in config", expect)
	}

	return rp
}

func expectRequiredProviderSource(t *testing.T, expect, actual string) {
	if expect != actual {
		t.Fatalf(
			"expected required provider source to be '%s', got '%s'",
			expect,
			actual,
		)
	}
}

func expectRequiredProviderVersion(
	t *testing.T,
	expect string,
	actual version.Constraints,
) {
	if expect == "" && actual == nil {
		return
	}

	if expect == "" {
		t.Fatal("expected required provider version NOT to be set")
	}

	if expect != actual.String() {
		t.Fatalf(
			"expected required provider version to be '%s', got '%s'",
			expect,
			actual,
		)
	}
}
