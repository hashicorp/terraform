// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Validate_badCount(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	})

	m := testModule(t, "validate-bad-count")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_badResource_reference(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	})

	m := testModule(t, "validate-bad-resource-count")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_badVar(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
					"num": {Type: cty.String, Optional: true},
				},
			},
		},
	})

	m := testModule(t, "validate-bad-var")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_varNoDefaultExplicitType(t *testing.T) {
	m := testModule(t, "validate-var-no-default-explicit-type")
	c, diags := NewContext(&ContextOpts{})
	if diags.HasErrors() {
		t.Fatalf("unexpected NewContext errors: %s", diags.Err())
	}

	// NOTE: This test has grown idiosyncratic because originally Terraform
	// would (optionally) check variables during validation, and then in
	// Terraform v0.12 we switched to checking variables during NewContext,
	// and now most recently we've switched to checking variables only during
	// planning because root variables are a plan option. Therefore this has
	// grown into a plan test rather than a validate test, but it lives on
	// here in order to make it easier to navigate through that history in
	// version control.
	_, diags = c.Plan(m, states.NewState(), DefaultPlanOpts)
	if !diags.HasErrors() {
		// Error should be: The input variable "maybe_a_map" has not been assigned a value.
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_computedVar(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		},
	}
	pt := testProvider("test")
	pt.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Computed: true},
						"value": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	m := testModule(t, "validate-computed-var")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"):  testProviderFuncFixed(p),
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(pt),
		},
	})

	p.ValidateProviderConfigFn = func(req providers.ValidateProviderConfigRequest) (resp providers.ValidateProviderConfigResponse) {
		val := req.Config.GetAttr("value")
		if val.IsKnown() {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("value isn't computed"))
		}

		return
	}

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
	if p.ConfigureProviderCalled {
		t.Fatal("Configure should not be called for provider")
	}
}

func TestContext2Validate_computedInFunction(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"attr": {Type: cty.Number, Optional: true},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"aws_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"optional_attr": {Type: cty.String, Optional: true},
						"computed":      {Type: cty.String, Computed: true},
					},
				},
			},
		},
	}

	m := testModule(t, "validate-computed-in-function")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

// Test that validate allows through computed counts. We do this and allow
// them to fail during "plan" since we can't know if the computed values
// can be realized during a plan.
func TestContext2Validate_countComputed(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"aws_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"compute": {Type: cty.String, Optional: true},
						"value":   {Type: cty.String, Computed: true},
					},
				},
			},
		},
	}

	m := testModule(t, "validate-count-computed")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_countNegative(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		},
	}
	m := testModule(t, "validate-count-negative")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_countVariable(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	m := testModule(t, "apply-count-variable")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_countVariableNoDefault(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-count-variable")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	c, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})
	assertNoDiagnostics(t, diags)

	_, diags = c.Plan(m, nil, &PlanOpts{})
	if !diags.HasErrors() {
		// Error should be: The input variable "foo" has not been assigned a value.
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_moduleBadOutput(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	m := testModule(t, "validate-bad-module-output")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_moduleGood(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	m := testModule(t, "validate-good-module")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_moduleBadResource(t *testing.T) {
	m := testModule(t, "validate-module-bad-rc")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ValidateResourceConfigResponse = &providers.ValidateResourceConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_moduleDepsShouldNotCycle(t *testing.T) {
	m := testModule(t, "validate-module-deps-cycle")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_moduleProviderVar(t *testing.T) {
	m := testModule(t, "validate-module-pc-vars")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ValidateProviderConfigFn = func(req providers.ValidateProviderConfigRequest) (resp providers.ValidateProviderConfigResponse) {
		if req.Config.GetAttr("foo").IsNull() {
			resp.Diagnostics = resp.Diagnostics.Append(errors.New("foo is null"))
		}
		return
	}

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_moduleProviderInheritUnused(t *testing.T) {
	m := testModule(t, "validate-module-pc-inherit-unused")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ValidateProviderConfigFn = func(req providers.ValidateProviderConfigRequest) (resp providers.ValidateProviderConfigResponse) {
		if req.Config.GetAttr("foo").IsNull() {
			resp.Diagnostics = resp.Diagnostics.Append(errors.New("foo is null"))
		}
		return
	}

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_orphans(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
						"num": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	m := testModule(t, "validate-good")

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		var diags tfdiags.Diagnostics
		if req.Config.GetAttr("foo").IsNull() {
			diags = diags.Append(errors.New("foo is not set"))
		}
		return providers.ValidateResourceConfigResponse{
			Diagnostics: diags,
		}
	}

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_providerConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-pc")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ValidateProviderConfigResponse = &providers.ValidateProviderConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate(m, nil)
	if len(diags) != 1 {
		t.Fatalf("wrong number of diagnostics %d; want %d", len(diags), 1)
	}
	if !strings.Contains(diags.Err().Error(), "bad") {
		t.Fatalf("bad: %s", diags.Err().Error())
	}
}

func TestContext2Validate_providerConfig_skippedEmpty(t *testing.T) {
	m := testModule(t, "validate-skipped-pc-empty")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ValidateProviderConfigResponse = &providers.ValidateProviderConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("should not be called")),
	}

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_providerConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-pc")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

// In this test there is a mismatch between the provider's fqn (hashicorp/test)
// and it's local name set in required_providers (arbitrary).
func TestContext2Validate_requiredProviderConfig(t *testing.T) {
	m := testModule(t, "validate-required-provider-config")
	p := testProvider("aws")

	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"required_attribute": {Type: cty.String, Required: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_provisionerConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-prov-conf")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	pr := simpleMockProvisioner()

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Provisioners: map[string]provisioners.Factory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	p.ValidateProviderConfigResponse = &providers.ValidateProviderConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_badResourceConnection(t *testing.T) {
	m := testModule(t, "validate-bad-resource-connection")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	pr := simpleMockProvisioner()

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Provisioners: map[string]provisioners.Factory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	diags := c.Validate(m, nil)
	t.Log(diags.Err())
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_badProvisionerConnection(t *testing.T) {
	m := testModule(t, "validate-bad-prov-connection")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	pr := simpleMockProvisioner()

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Provisioners: map[string]provisioners.Factory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	diags := c.Validate(m, nil)
	t.Log(diags.Err())
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_provisionerConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-prov-conf")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	pr := simpleMockProvisioner()
	pr.ValidateProvisionerConfigFn = func(req provisioners.ValidateProvisionerConfigRequest) provisioners.ValidateProvisionerConfigResponse {
		var diags tfdiags.Diagnostics
		if req.Config.GetAttr("test_string").IsNull() {
			diags = diags.Append(errors.New("test_string is not set"))
		}
		return provisioners.ValidateProvisionerConfigResponse{
			Diagnostics: diags,
		}
	}

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Provisioners: map[string]provisioners.Factory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_requiredVar(t *testing.T) {
	m := testModule(t, "validate-required-var")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ami": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	c, diags := NewContext(&ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})
	assertNoDiagnostics(t, diags)

	// NOTE: This test has grown idiosyncratic because originally Terraform
	// would (optionally) check variables during validation, and then in
	// Terraform v0.12 we switched to checking variables during NewContext,
	// and now most recently we've switched to checking variables only during
	// planning because root variables are a plan option. Therefore this has
	// grown into a plan test rather than a validate test, but it lives on
	// here in order to make it easier to navigate through that history in
	// version control.
	_, diags = c.Plan(m, states.NewState(), DefaultPlanOpts)
	if !diags.HasErrors() {
		// Error should be: The input variable "foo" has not been assigned a value.
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_resourceConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-rc")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ValidateResourceConfigResponse = &providers.ValidateResourceConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_resourceConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-rc")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_tainted(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
						"num": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	m := testModule(t, "validate-good")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	p.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		var diags tfdiags.Diagnostics
		if req.Config.GetAttr("foo").IsNull() {
			diags = diags.Append(errors.New("foo is not set"))
		}
		return providers.ValidateResourceConfigResponse{
			Diagnostics: diags,
		}
	}

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_targetedDestroy(t *testing.T) {
	m := testModule(t, "validate-targeted")
	p := testProvider("aws")
	pr := simpleMockProvisioner()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
						"num": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	testSetResourceInstanceCurrent(root, "aws_instance.foo", `{"id":"i-bcd345"}`, `provider["registry.terraform.io/hashicorp/aws"]`)
	testSetResourceInstanceCurrent(root, "aws_instance.bar", `{"id":"i-abc123"}`, `provider["registry.terraform.io/hashicorp/aws"]`)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Provisioners: map[string]provisioners.Factory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_varRefUnknown(t *testing.T) {
	m := testModule(t, "validate-variable-ref")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	var value cty.Value
	p.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		value = req.Config.GetAttr("foo")
		return providers.ValidateResourceConfigResponse{}
	}

	c.Validate(m, nil)

	// Input variables are always unknown during the validate walk, because
	// we're checking for validity of all possible input values. Validity
	// against specific input values is checked during the plan walk.
	if !value.RawEquals(cty.UnknownVal(cty.String)) {
		t.Fatalf("bad: %#v", value)
	}
}

// Module variables weren't being interpolated during Validate phase.
// related to https://github.com/hashicorp/terraform/issues/5322
func TestContext2Validate_interpolateVar(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "input-interpolate-var")
	p := testProvider("null")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"template_file": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"template": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("template"): testProviderFuncFixed(p),
		},
		UIInput: input,
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

// When module vars reference something that is actually computed, this
// shouldn't cause validation to fail.
func TestContext2Validate_interpolateComputedModuleVarDef(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "validate-computed-module-var-ref")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"attr": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		UIInput: input,
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

// Computed values are lost when a map is output from a module
func TestContext2Validate_interpolateMap(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "issue-9549")
	p := testProvider("template")

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("template"): testProviderFuncFixed(p),
		},
		UIInput: input,
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_varSensitive(t *testing.T) {
	// Smoke test through validate where a variable has sensitive applied
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "foo" {
  default = "xyz"
  sensitive = true
}

variable "bar" {
  sensitive = true
}

data "aws_data_source" "bar" {
  foo = var.bar
}

resource "aws_instance" "foo" {
  foo = var.foo
}
`,
	})

	p := testProvider("aws")
	p.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		// Providers receive unmarked values
		if got, want := req.Config.GetAttr("foo"), cty.UnknownVal(cty.String); !got.RawEquals(want) {
			t.Fatalf("wrong value for foo\ngot:  %#v\nwant: %#v", got, want)
		}
		return providers.ValidateResourceConfigResponse{}
	}
	p.ValidateDataResourceConfigFn = func(req providers.ValidateDataResourceConfigRequest) (resp providers.ValidateDataResourceConfigResponse) {
		if got, want := req.Config.GetAttr("foo"), cty.UnknownVal(cty.String); !got.RawEquals(want) {
			t.Fatalf("wrong value for foo\ngot:  %#v\nwant: %#v", got, want)
		}
		return providers.ValidateDataResourceConfigResponse{}
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if !p.ValidateResourceConfigCalled {
		t.Fatal("expected ValidateResourceConfigFn to be called")
	}

	if !p.ValidateDataResourceConfigCalled {
		t.Fatal("expected ValidateDataSourceConfigFn to be called")
	}
}

func TestContext2Validate_invalidOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
data "aws_data_source" "name" {}

output "out" {
  value = "${data.aws_data_source.name.missing}"
}`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Unsupported attribute: This object does not have an attribute named "missing"
	if got, want := diags.Err().Error(), "Unsupported attribute"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_invalidModuleOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"child/main.tf": `
data "aws_data_source" "name" {}

output "out" {
  value = "${data.aws_data_source.name.missing}"
}`,
		"main.tf": `
module "child" {
  source = "./child"
}

resource "aws_instance" "foo" {
  foo = "${module.child.out}"
}`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Unsupported attribute: This object does not have an attribute named "missing"
	if got, want := diags.Err().Error(), "Unsupported attribute"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_sensitiveRootModuleOutput(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"child/main.tf": `
variable "foo" {
  default = "xyz"
  sensitive = true
}

output "out" {
  value = var.foo
}`,
		"main.tf": `
module "child" {
  source = "./child"
}

output "root" {
  value = module.child.out
  sensitive = true
}`,
	})

	ctx := testContext2(t, &ContextOpts{})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
}

func TestContext2Validate_legacyResourceCount(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "test" {}

output "out" {
  value = aws_instance.test.count
}`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Invalid resource count attribute: The special "count" attribute is no longer supported after Terraform v0.12. Instead, use length(aws_instance.test) to count resource instances.
	if got, want := diags.Err().Error(), "Invalid resource count attribute:"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_invalidModuleRef(t *testing.T) {
	// This test is verifying that we properly validate and report on references
	// to modules that are not declared, since we were missing some validation
	// here in early 0.12.0 alphas that led to a panic.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
output "out" {
  # Intentionally referencing undeclared module to ensure error
  value = module.foo
}`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Reference to undeclared module: No module call named "foo" is declared in the root module.
	if got, want := diags.Err().Error(), "Reference to undeclared module:"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_invalidModuleOutputRef(t *testing.T) {
	// This test is verifying that we properly validate and report on references
	// to modules that are not declared, since we were missing some validation
	// here in early 0.12.0 alphas that led to a panic.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
output "out" {
  # Intentionally referencing undeclared module to ensure error
  value = module.foo.bar
}`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Reference to undeclared module: No module call named "foo" is declared in the root module.
	if got, want := diags.Err().Error(), "Reference to undeclared module:"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_invalidDependsOnResourceRef(t *testing.T) {
	// This test is verifying that we raise an error if depends_on
	// refers to something that doesn't exist in configuration.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "bar" {
  depends_on = [test_resource.nonexistant]
}
`,
	})

	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Reference to undeclared module: No module call named "foo" is declared in the root module.
	if got, want := diags.Err().Error(), "Reference to undeclared resource:"; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_invalidResourceIgnoreChanges(t *testing.T) {
	// This test is verifying that we raise an error if ignore_changes
	// refers to something that can be statically detected as not conforming
	// to the resource type schema.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "bar" {
  lifecycle {
    ignore_changes = [does_not_exist_in_schema]
  }
}
`,
	})

	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Reference to undeclared module: No module call named "foo" is declared in the root module.
	if got, want := diags.Err().Error(), `no argument, nested block, or exported attribute named "does_not_exist_in_schema"`; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_variableCustomValidationsFail(t *testing.T) {
	// This test is for custom validation rules associated with root module
	// variables, and specifically that we handle the situation where the
	// given value is invalid in a child module.
	m := testModule(t, "validate-variable-custom-validations-child")

	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := diags.Err().Error(), `Invalid value for variable: Value must not be "nope".`; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_variableCustomValidationsRoot(t *testing.T) {
	// This test is for custom validation rules associated with root module
	// variables, and specifically that we handle the situation where their
	// values are unknown during validation, skipping the validation check
	// altogether. (Root module variables are never known during validation.)
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "test" {
  type = string

  validation {
	condition     = var.test != "nope"
	error_message = "Value must not be \"nope\"."
  }
}
`,
	})

	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error\ngot: %s", diags.Err().Error())
	}
}

func TestContext2Validate_expandModules(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod1" {
  for_each = toset(["a", "b"])
  source = "./mod"
}

module "mod2" {
  for_each = module.mod1
  source = "./mod"
  input = module.mod1["a"].out
}

module "mod3" {
  count = length(module.mod2)
  source = "./mod"
}
`,
		"mod/main.tf": `
resource "aws_instance" "foo" {
}

output "out" {
  value = 1
}

variable "input" {
  type = number
  default = 0
}

module "nested" {
  count = 2
  source = "./nested"
  input = count.index
}
`,
		"mod/nested/main.tf": `
variable "input" {
}

resource "aws_instance" "foo" {
  count = var.input
}
`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_expandModulesInvalidCount(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod1" {
  count = -1
  source = "./mod"
}
`,
		"mod/main.tf": `
resource "aws_instance" "foo" {
}
`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := diags.Err().Error(), `Invalid count argument`; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_expandModulesInvalidForEach(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod1" {
  for_each = ["a", "b"]
  source = "./mod"
}
`,
		"mod/main.tf": `
resource "aws_instance" "foo" {
}
`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	if got, want := diags.Err().Error(), `Invalid for_each argument`; !strings.Contains(got, want) {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}

func TestContext2Validate_expandMultipleNestedModules(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "modA" {
  for_each = {
    first = "m"
	second = "n"
  }
  source = "./modA"
}
`,
		"modA/main.tf": `
locals {
  m = {
    first = "m"
	second = "n"
  }
}

module "modB" {
  for_each = local.m
  source = "./modB"
  y = each.value
}

module "modC" {
  for_each = local.m
  source = "./modC"
  x = module.modB[each.key].out
  y = module.modB[each.key].out
}

`,
		"modA/modB/main.tf": `
variable "y" {
  type = string
}

resource "aws_instance" "foo" {
  foo = var.y
}

output "out" {
  value = aws_instance.foo.id
}
`,
		"modA/modC/main.tf": `
variable "x" {
  type = string
}

variable "y" {
  type = string
}

resource "aws_instance" "foo" {
  foo = var.x
}

output "out" {
  value = var.y
}
`,
	})

	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_invalidModuleDependsOn(t *testing.T) {
	// validate module and output depends_on
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod1" {
  source = "./mod"
  depends_on = [resource_foo.bar.baz]
}

module "mod2" {
  source = "./mod"
  depends_on = [resource_foo.bar.baz]
}
`,
		"mod/main.tf": `
output "out" {
  value = "foo"
}
`,
	})

	diags := testContext2(t, &ContextOpts{}).Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}

	if len(diags) != 2 {
		t.Fatalf("wanted 2 diagnostic errors, got %q", diags)
	}

	for _, d := range diags {
		des := d.Description().Summary
		if !strings.Contains(des, "Invalid depends_on reference") {
			t.Fatalf(`expected "Invalid depends_on reference", got %q`, des)
		}
	}
}

func TestContext2Validate_invalidOutputDependsOn(t *testing.T) {
	// validate module and output depends_on
	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "mod1" {
  source = "./mod"
}

output "out" {
  value = "bar"
  depends_on = [resource_foo.bar.baz]
}
`,
		"mod/main.tf": `
output "out" {
  value = "bar"
  depends_on = [resource_foo.bar.baz]
}
`,
	})

	diags := testContext2(t, &ContextOpts{}).Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}

	if len(diags) != 2 {
		t.Fatalf("wanted 2 diagnostic errors, got %q", diags)
	}

	for _, d := range diags {
		des := d.Description().Summary
		if !strings.Contains(des, "Invalid depends_on reference") {
			t.Fatalf(`expected "Invalid depends_on reference", got %q`, des)
		}
	}
}

func TestContext2Validate_rpcDiagnostics(t *testing.T) {
	// validate module and output depends_on
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
}
`,
	})

	p := testProvider("test")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Computed: true},
					},
				},
			},
		},
	}

	p.ValidateResourceConfigResponse = &providers.ValidateResourceConfigResponse{
		Diagnostics: tfdiags.Diagnostics(nil).Append(tfdiags.SimpleWarning("don't frobble")),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if len(diags) == 0 {
		t.Fatal("expected warnings")
	}

	for _, d := range diags {
		des := d.Description().Summary
		if !strings.Contains(des, "frobble") {
			t.Fatalf(`expected frobble, got %q`, des)
		}
	}
}

func TestContext2Validate_sensitiveProvisionerConfig(t *testing.T) {
	m := testModule(t, "validate-sensitive-provisioner-config")
	p := testProvider("aws")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"aws_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	pr := simpleMockProvisioner()

	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
		Provisioners: map[string]provisioners.Factory{
			"test": testProvisionerFuncFixed(pr),
		},
	})

	pr.ValidateProvisionerConfigFn = func(r provisioners.ValidateProvisionerConfigRequest) provisioners.ValidateProvisionerConfigResponse {
		if r.Config.ContainsMarked() {
			t.Errorf("provisioner config contains marked values")
		}
		return pr.ValidateProvisionerConfigResponse
	}

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
	if !pr.ValidateProvisionerConfigCalled {
		t.Fatal("ValidateProvisionerConfig not called")
	}
}

func TestContext2Plan_validateMinMaxDynamicBlock(t *testing.T) {
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"things": {
						Type:     cty.List(cty.String),
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {Type: cty.String, Optional: true},
							},
						},
						Nesting:  configschema.NestingList,
						MinItems: 2,
						MaxItems: 3,
					},
				},
			},
		},
	})

	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_instance" "a" {
  // MinItems 2
  foo {
    bar = "a"
  }
  foo {
    bar = "b"
  }
}

resource "test_instance" "b" {
  // one dymamic block can satisfy MinItems of 2
  dynamic "foo" {
	for_each = test_instance.a.things
	content {
	  bar = foo.value
	}
  }
}

resource "test_instance" "c" {
  // we may have more than MaxItems dynamic blocks when they are unknown
  foo {
    bar = "b"
  }
  dynamic "foo" {
    for_each = test_instance.a.things
    content {
      bar = foo.value
    }
  }
  dynamic "foo" {
    for_each = test_instance.a.things
    content {
      bar = "${foo.value}-2"
    }
  }
  dynamic "foo" {
    for_each = test_instance.b.things
    content {
      bar = foo.value
    }
  }
}
`})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_passInheritedProvider(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
terraform {
  required_providers {
	test = {
	  source = "hashicorp/test"
	}
  }
}

module "first" {
  source = "./first"
  providers = {
    test = test
  }
}
`,

		// This module does not define a config for the test provider, but we
		// should be able to pass whatever the implied config is to a child
		// module.
		"first/main.tf": `
terraform {
  required_providers {
    test = {
	  source = "hashicorp/test"
    }
  }
}

module "second" {
  source = "./second"
  providers = {
	test.alias = test
  }
}`,

		"first/second/main.tf": `
terraform {
  required_providers {
    test = {
	  source = "hashicorp/test"
      configuration_aliases = [test.alias]
    }
  }
}

resource "test_object" "t" {
  provider = test.alias
}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Plan_lookupMismatchedObjectTypes(t *testing.T) {
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"things": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
		},
	})

	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "items" {
  type = list(string)
  default = []
}

resource "test_instance" "a" {
  for_each = length(var.items) > 0 ? { default = {} } : {}
}

output "out" {
  // Strictly speaking, this expression is incorrect because the map element
  // type is a different type from the default value, and the lookup
  // implementation expects to be able to convert the default to match the
  // element type.
  // There are two reasons this works which we need to maintain for
  // compatibility. First during validation the 'test_instance.a' expression
  // only returns a dynamic value, preventing any type comparison. Later during
  // plan and apply 'test_instance.a' is an object and not a map, and the
  // lookup implementation skips the type comparison when the keys are known
  // statically.
  value = lookup(test_instance.a, "default", { id = null })["id"]
}
`})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_nonNullableVariableDefaultValidation(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
 module "first" {
   source = "./mod"
   input = null
 }
 `,

		"mod/main.tf": `
 variable "input" {
   type        = string
   default     = "default"
   nullable    = false

   // Validation expressions should receive the default with nullable=false and
   // a null input.
   validation {
     condition     = var.input != null
     error_message = "Input cannot be null!"
   }
 }
 `,
	})

	ctx := testContext2(t, &ContextOpts{})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_precondition_good(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "input" {
  type    = string
  default = "foo"
}

resource "aws_instance" "test" {
  foo = var.input

  lifecycle {
    precondition {
      condition     = length(var.input) > 0
      error_message = "Input cannot be empty."
    }
  }
}
 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_precondition_badCondition(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "input" {
  type    = string
  default = "foo"
}

resource "aws_instance" "test" {
  foo = var.input

  lifecycle {
    precondition {
      condition     = length(one(var.input)) == 1
      error_message = "You can't do that."
    }
  }
}
 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
	if got, want := diags.Err().Error(), "Invalid function argument"; !strings.Contains(got, want) {
		t.Errorf("unexpected error.\ngot: %s\nshould contain: %q", got, want)
	}
}

func TestContext2Validate_precondition_badErrorMessage(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "input" {
  type    = string
  default = "foo"
}

resource "aws_instance" "test" {
  foo = var.input

  lifecycle {
    precondition {
      condition     = var.input != "foo"
      error_message = "This is a bad use of a function: ${one(var.input)}."
    }
  }
}
 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
	if got, want := diags.Err().Error(), "Invalid function argument"; !strings.Contains(got, want) {
		t.Errorf("unexpected error.\ngot: %s\nshould contain: %q", got, want)
	}
}

func TestContext2Validate_postcondition_good(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "test" {
  foo = "foo"

  lifecycle {
    postcondition {
      condition     = length(self.foo) > 0
      error_message = "Input cannot be empty."
    }
  }
}
 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_postcondition_badCondition(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	// This postcondition's condition expression does not refer to self, which
	// is unrealistic. This is because at the time of writing the test, self is
	// always an unknown value of dynamic type during validation. As a result,
	// validation of conditions which refer to resource arguments is not
	// possible until plan time. For now we exercise the code by referring to
	// an input variable.
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "input" {
  type    = string
  default = "foo"
}

resource "aws_instance" "test" {
  foo = var.input

  lifecycle {
    postcondition {
      condition     = length(one(var.input)) == 1
      error_message = "You can't do that."
    }
  }
}
 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
	if got, want := diags.Err().Error(), "Invalid function argument"; !strings.Contains(got, want) {
		t.Errorf("unexpected error.\ngot: %s\nshould contain: %q", got, want)
	}
}

func TestContext2Validate_postcondition_badErrorMessage(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "test" {
  foo = "foo"

  lifecycle {
    postcondition {
      condition     = self.foo != "foo"
      error_message = "This is a bad use of a function: ${one("foo")}."
    }
  }
}
 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
	if got, want := diags.Err().Error(), "Invalid function argument"; !strings.Contains(got, want) {
		t.Errorf("unexpected error.\ngot: %s\nshould contain: %q", got, want)
	}
}

func TestContext2Validate_precondition_count(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  foos = ["bar", "baz"]
}

resource "aws_instance" "test" {
  count = 3
  foo = local.foos[count.index]

  lifecycle {
    precondition {
      condition     = count.index < length(local.foos)
      error_message = "Insufficient foos."
    }
  }
}
 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_postcondition_forEach(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	})
	m := testModuleInline(t, map[string]string{
		"main.tf": `
locals {
  foos = toset(["bar", "baz", "boop"])
}

resource "aws_instance" "test" {
  for_each = local.foos
  foo = "foo"

  lifecycle {
    postcondition {
      condition     = length(each.value) == 3
      error_message = "Short foo required, not \"${each.key}\"."
    }
  }
}
 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_deprecatedAttr(t *testing.T) {
	p := testProvider("aws")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true, Deprecated: true},
				},
			},
		},
	})
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "test" {
}
locals {
  deprecated = aws_instance.test.foo
}

 `,
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, nil)
	warn := diags.ErrWithWarnings().Error()
	if !strings.Contains(warn, `The attribute "foo" is deprecated`) {
		t.Fatalf("expected deprecated warning, got: %q\n", warn)
	}
}

func TestContext2Validate_unknownForEach(t *testing.T) {
	p := testProvider("aws")
	m := testModuleInline(t, map[string]string{
		"main.tf": `
resource "aws_instance" "test" {
}

locals {
  follow = {
    (aws_instance.test.id): "follow"
  }
}

resource "aws_instance" "follow" {
  for_each = local.follow
}
 `,
	})
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	diags := c.Validate(m, nil)
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
}

func TestContext2Validate_providerContributedFunctions(t *testing.T) {
	mockProvider := func() *testing_provider.MockProvider {
		p := testProvider("test")
		p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
			Functions: map[string]providers.FunctionDecl{
				"count_e": {
					ReturnType: cty.Number,
					Parameters: []providers.FunctionParam{
						{
							Name: "string",
							Type: cty.String,
						},
					},
				},
			},
		}
		p.CallFunctionFn = func(req providers.CallFunctionRequest) (resp providers.CallFunctionResponse) {
			if req.FunctionName != "count_e" {
				resp.Err = fmt.Errorf("incorrect function name %q", req.FunctionName)
				return resp
			}
			if len(req.Arguments) != 1 {
				resp.Err = fmt.Errorf("wrong number of arguments %d", len(req.Arguments))
				return resp
			}
			if req.Arguments[0].Type() != cty.String {
				resp.Err = fmt.Errorf("wrong argument type %#v", req.Arguments[0].Type())
				return resp
			}
			if !req.Arguments[0].IsKnown() {
				resp.Err = fmt.Errorf("argument is unknown")
				return resp
			}
			if req.Arguments[0].IsNull() {
				resp.Err = fmt.Errorf("argument is null")
				return resp
			}

			str := req.Arguments[0].AsString()
			count := strings.Count(str, "e")
			resp.Result = cty.NumberIntVal(int64(count))
			return resp
		}
		return p
	}

	t.Run("valid", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		test = {
			source = "hashicorp/test"
		}
	}
}
locals {
	result = provider::test::count_e("cheese")
}
output "result" {
	value = local.result
	precondition {
		condition     = (local.result == 3)
		error_message = "Wrong number of Es in my cheese."
	}
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		diags := ctx.Validate(m, nil)
		if diags.HasErrors() {
			t.Fatal(diags.ErrWithWarnings())
		}

		if !p.CallFunctionCalled {
			t.Fatal("CallFunction was not called")
		}
	})
	t.Run("wrong name", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		test = {
			source = "hashicorp/test"
		}
	}
}
output "result" {
	value = provider::test::cout_e("cheese")
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		diags := ctx.Validate(m, nil)
		if p.CallFunctionCalled {
			t.Error("CallFunction was called, but should not have been")
		}
		if !diags.HasErrors() {
			t.Fatal("unexpected success")
		}
		if got, want := diags.Err().Error(), `Unknown provider function: The function "cout_e" is not available from the provider "test"`; !strings.Contains(got, want) {
			t.Errorf("wrong error message\nwant substring: %s\ngot: %s", want, got)
		}

	})
	t.Run("wrong namespace", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		test = {
			source = "hashicorp/test"
		}
	}
}
output "result" {
	value = provider::toast::count_e("cheese")
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		diags := ctx.Validate(m, nil)
		if p.CallFunctionCalled {
			t.Error("CallFunction was called, but should not have been")
		}
		if !diags.HasErrors() {
			t.Fatal("unexpected success")
		}
		if got, want := diags.Err().Error(), `Unknown provider function: There is no function named "provider::toast::count_e`; !strings.Contains(got, want) {
			t.Errorf("wrong error message\nwant substring: %s\ngot: %s", want, got)
		}
	})
	t.Run("wrong argument type", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		test = {
			source = "hashicorp/test"
		}
	}
}
output "result" {
	value = provider::test::count_e([])
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		diags := ctx.Validate(m, nil)
		if p.CallFunctionCalled {
			t.Error("CallFunction was called, but should not have been")
		}
		if !diags.HasErrors() {
			t.Fatal("unexpected success")
		}
		if got, want := diags.Err().Error(), "Invalid function argument: Invalid value for \"string\" parameter: string required."; !strings.Contains(got, want) {
			t.Errorf("wrong error message\nwant substring: %s\ngot: %s", want, got)
		}
	})
	t.Run("insufficient arguments", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		test = {
			source = "hashicorp/test"
		}
	}
}
output "result" {
	value = provider::test::count_e()
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		diags := ctx.Validate(m, nil)
		if p.CallFunctionCalled {
			t.Error("CallFunction was called, but should not have been")
		}
		if !diags.HasErrors() {
			t.Fatal("unexpected success")
		}
		if got, want := diags.Err().Error(), "Not enough function arguments: Function \"provider::test::count_e\" expects 1 argument(s). Missing value for \"string\"."; !strings.Contains(got, want) {
			t.Errorf("wrong error message\nwant substring: %s\ngot: %s", want, got)
		}
	})
	t.Run("too many arguments", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		test = {
			source = "hashicorp/test"
		}
	}
}
output "result" {
	value = provider::test::count_e("cheese", "louise")
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		diags := ctx.Validate(m, nil)
		if p.CallFunctionCalled {
			t.Error("CallFunction was called, but should not have been")
		}
		if !diags.HasErrors() {
			t.Fatal("unexpected success")
		}
		if got, want := diags.Err().Error(), "Too many function arguments: Function \"provider::test::count_e\" expects only 1 argument(s)."; !strings.Contains(got, want) {
			t.Errorf("wrong error message\nwant substring: %s\ngot: %s", want, got)
		}
	})
	t.Run("unexpected null argument", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		test = {
			source = "hashicorp/test"
		}
	}
}
output "result" {
	value = provider::test::count_e(null)
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		diags := ctx.Validate(m, nil)
		if p.CallFunctionCalled {
			t.Error("CallFunction was called, but should not have been")
		}
		if !diags.HasErrors() {
			t.Fatal("unexpected success")
		}
		if got, want := diags.Err().Error(), "Invalid function argument: Invalid value for \"string\" parameter: argument must not be null."; !strings.Contains(got, want) {
			t.Errorf("wrong error message\nwant substring: %s\ngot: %s", want, got)
		}
	})
	t.Run("unhandled unknown argument", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		test = {
			source = "hashicorp/test"
		}
	}
}
output "result" {
	value = provider::test::count_e(timestamp())
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		// For this case, validation should succeed without calling the
		// function yet, because the function doesn't declare that it handles
		// unknown values and so we must defer validation until a later phase.
		diags := ctx.Validate(m, nil)
		if p.CallFunctionCalled {
			t.Error("CallFunction was called, but should not have been")
		}
		if diags.HasErrors() {
			t.Fatal(diags.ErrWithWarnings())
		}
	})
	t.Run("provider not declared", func(t *testing.T) {
		m := testModuleInline(t, map[string]string{
			"main.tf": `
terraform {
	required_providers {
		# Intentionally no declaration of local name "test" here
	}
}
output "result" {
	value = provider::test::count_e("cheese")
}
`,
		})

		p := mockProvider()
		ctx := testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		})

		diags := ctx.Validate(m, nil)
		if p.CallFunctionCalled {
			t.Error("CallFunction was called, but should not have been")
		}
		if !diags.HasErrors() {
			t.Fatal("unexpected success")
		}
		// Module author must declare a provider requirement in order to
		// import a provider's functions.
		if got, want := diags.Err().Error(), `Unknown provider function: There is no function named "provider::test::count_e"`; !strings.Contains(got, want) {
			t.Errorf("wrong error message\nwant substring: %s\ngot: %s", want, got)
		}
	})
}

func TestContextValidate_externalProviders(t *testing.T) {

	m := testModuleInline(t, map[string]string{
		"main.tf": `
terraform {
  required_providers {
    bar = {
      source = "hashicorp/bar"
	}
  }
}

provider "bar" {}

resource "bar_instance" "test" {
  foo = "foo" # should be an int
}
`,
	})

	mustNotConfigure := func(providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
		return providers.ConfigureProviderResponse{
			Diagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Pre-configured provider was reconfigured by the modules runtime",
					"An externally-configured provider should not have its ConfigureProvider function called during planning.",
				),
			},
		}
	}

	providerAddr := addrs.NewDefaultProvider("bar")
	providerConfigAddr := addrs.RootProviderConfig{
		Provider: providerAddr,
	}

	provider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						// We have a required attribute that is not set, we're
						// expecting this to not matter as we shouldn't validate
						// the provider configuration as we're using an external
						// provider.
						"required": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
			ResourceTypes: map[string]providers.Schema{
				"bar_instance": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							// We should still validate this attribute as being
							// incorrect, even though we have an external
							// provider.
							"foo": {
								Type:     cty.Number,
								Required: true,
							},
						},
					},
				},
			},
		},
		ConfigureProviderFn: mustNotConfigure,
	}

	ctx, diags := NewContext(&ContextOpts{
		PreloadedProviderSchemas: map[addrs.Provider]providers.ProviderSchema{
			providerAddr: *provider.GetProviderSchemaResponse,
		},
	})
	assertNoDiagnostics(t, diags)

	// Many of the MockProvider methods check for this, so we'll set it to be
	// true externally.
	provider.ConfigureProviderCalled = true

	diags = ctx.Validate(m, &ValidateOpts{
		ExternalProviders: map[addrs.RootProviderConfig]providers.Interface{
			providerConfigAddr: provider,
		},
	})

	// We should have exactly one diagnostic, stating there was an error in the
	// resource. But nothing complaining about the provider itself.

	if len(diags) != 1 {
		t.Fatalf("expected exactly one diagnostic, got %d", len(diags))
	}

	if diff := cmp.Diff(diags[0].Description().Summary, "Incorrect attribute value type"); len(diff) > 0 {
		t.Errorf("unexpected diagnostic summary: %s", diff)
	}
	if diff := cmp.Diff(diags[0].Description().Detail, "Inappropriate value for attribute \"foo\": a number is required."); len(diff) > 0 {
		t.Errorf("unexpected diagnostic detail: %s", diff)
	}
}

func TestContext2Validate_providerSchemaError(t *testing.T) {
	// validate module and output depends_on
	m := testModuleInline(t, map[string]string{
		"main.tf": `
terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

output "foo" {
	value = provider::test::func("foo")
}
`,
	})

	p := testProvider("test")
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Computed: true},
					},
				},
			},
		},
		Diagnostics: tfdiags.Diagnostics(nil).Append(errors.New("schema problem!")),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})
	diags := ctx.Validate(m, nil)
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}

	// while the function in the config doesn't exist, we should not have gotten
	// that far and stopped at the schema error first
	for _, d := range diags {
		if detail := d.Description().Detail; !strings.Contains(detail, "schema problem!") {
			t.Errorf("unexpected error: %s", detail)
		}
	}
}

func TestContext2Validate_ephemeralOutput_root(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "foo" {
  ephemeral = true
  default   = "foo"
}
output "test" {
  ephemeral = true
  value     = var.foo
}
`,
	})

	ctx := testContext2(t, &ContextOpts{})
	diags := ctx.Validate(m, &ValidateOpts{})
	var wantDiags tfdiags.Diagnostics
	wantDiags = wantDiags.Append(
		&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Ephemeral output not allowed",
			Detail:   "Ephemeral outputs are not allowed in context of a root module",
			Subject: &hcl.Range{
				Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
				Start:    hcl.Pos{Line: 6, Column: 1, Byte: 59},
				End:      hcl.Pos{Line: 6, Column: 14, Byte: 72},
			},
		},
	)
	assertDiagnosticsMatch(t, diags, wantDiags)
}

func TestContext2Validate_ephemeralOutput_child(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"child/main.tf": `
variable "child-eph" {
  ephemeral = true
}
output "out" {
  ephemeral = true
  value     = var.child-eph
}`,
		"main.tf": `
variable "eph" {
  ephemeral = true
  default   = "foo"
}

module "child" {
  source    = "./child"
  child-eph = var.eph
}
`,
	})

	ctx := testContext2(t, &ContextOpts{})
	diags := ctx.Validate(m, &ValidateOpts{})
	assertNoDiagnostics(t, diags)
}

func TestContext2Validate_deprecated_output(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"mod/main.tf": `
output "old" {
    deprecated = "Please stop using this"
    value = "old"
}

output "old-and-unused" {
    deprecated = "This should not show up in the errors, we are not using it"
    value = "old"
}

output "new" {
    value = "foo"
}
`,
		"mod2/main.tf": `
variable "input" {
	type = string
}
`,
		"main.tf": `
module "mod" {
    source = "./mod"
}

resource "test_resource" "test" {
    attr = module.mod.old
}

resource "test_resource" "test2" {
    attr = module.mod.new
}

resource "test_resource" "test3" {
    attr = module.mod.old
}

output "test_output" {
	value = module.mod.old
}

module "mod2" {
	source = "./mod2"

	input = module.mod.old
}
`,
	})

	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	})

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	diags := ctx.Validate(m, &ValidateOpts{})
	var expectedDiags tfdiags.Diagnostics
	expectedDiags = expectedDiags.Append(
		&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Usage of deprecated output",
			Detail:   "Please stop using this",
			Subject: &hcl.Range{
				Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
				Start:    hcl.Pos{Line: 7, Column: 12, Byte: 85},
				End:      hcl.Pos{Line: 7, Column: 26, Byte: 99},
			},
		},
		&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Usage of deprecated output",
			Detail:   "Please stop using this",
			Subject: &hcl.Range{
				Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
				Start:    hcl.Pos{Line: 15, Column: 12, Byte: 213},
				End:      hcl.Pos{Line: 15, Column: 26, Byte: 227},
			},
		},
		&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Usage of deprecated output",
			Detail:   "Please stop using this",
			Subject: &hcl.Range{
				Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
				Start:    hcl.Pos{Line: 19, Column: 10, Byte: 263},
				End:      hcl.Pos{Line: 19, Column: 24, Byte: 277},
			},
		},
		&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Usage of deprecated output",
			Detail:   "Please stop using this",
			Subject: &hcl.Range{
				Filename: filepath.Join(m.Module.SourceDir, "main.tf"),
				Start:    hcl.Pos{Line: 25, Column: 10, Byte: 326},
				End:      hcl.Pos{Line: 25, Column: 24, Byte: 340},
			},
		},
	)

	assertDiagnosticsMatch(t, diags, expectedDiags)
}
