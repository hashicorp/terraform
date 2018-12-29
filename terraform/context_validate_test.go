package terraform

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestContext2Validate_badCount(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}

	m := testModule(t, "validate-bad-count")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_badVar(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
					"num": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "validate-bad-var")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_varMapOverrideOld(t *testing.T) {
	m := testModule(t, "validate-module-pc-vars")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo.foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCaller,
			},
		},
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_varNoDefaultExplicitType(t *testing.T) {
	m := testModule(t, "validate-var-no-default-explicit-type")
	c := testContext2(t, &ContextOpts{
		Config: m,
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_computedVar(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"value": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}
	pt := testProvider("test")
	pt.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "validate-computed-var")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws":  testProviderFuncFixed(p),
				"test": testProviderFuncFixed(pt),
			},
		),
	})

	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		if !c.IsComputed("value") {
			return nil, []error{fmt.Errorf("value isn't computed")}
		}

		return nil, c.CheckSet([]string{"value"})
	}

	p.ConfigureFn = func(c *ResourceConfig) error {
		return fmt.Errorf("Configure should not be called for provider")
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_computedInFunction(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {Type: cty.Number, Optional: true},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"optional_attr": {Type: cty.String, Optional: true},
					"computed":      {Type: cty.String, Computed: true},
				},
			},
		},
	}

	m := testModule(t, "validate-computed-in-function")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

// Test that validate allows through computed counts. We do this and allow
// them to fail during "plan" since we can't know if the computed values
// can be realized during a plan.
func TestContext2Validate_countComputed(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
		DataSources: map[string]*configschema.Block{
			"aws_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"compute": {Type: cty.String, Optional: true},
					"value":   {Type: cty.String, Computed: true},
				},
			},
		},
	}

	m := testModule(t, "validate-count-computed")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_countNegative(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}

	m := testModule(t, "validate-count-negative")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_countVariable(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "apply-count-variable")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_countVariableNoDefault(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "validate-count-variable")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_moduleBadOutput(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "validate-bad-module-output")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_moduleGood(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "validate-good-module")
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_moduleBadResource(t *testing.T) {
	m := testModule(t, "validate-module-bad-rc")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ValidateResourceTypeConfigResponse = providers.ValidateResourceTypeConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_moduleDepsShouldNotCycle(t *testing.T) {
	m := testModule(t, "validate-module-deps-cycle")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_moduleProviderVar(t *testing.T) {
	m := testModule(t, "validate-module-pc-vars")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"provider_var": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCaller,
			},
		},
	})

	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		return nil, c.CheckSet([]string{"foo"})
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_moduleProviderInheritUnused(t *testing.T) {
	m := testModule(t, "validate-module-pc-inherit-unused")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		return nil, c.CheckSet([]string{"foo"})
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_orphans(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
					"num": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "validate-good")
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	})
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		var diags tfdiags.Diagnostics
		if req.Config.GetAttr("foo").IsNull() {
			diags.Append(errors.New("foo is not set"))
		}
		return providers.ValidateResourceTypeConfigResponse{
			Diagnostics: diags,
		}
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_providerConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-pc")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.PrepareProviderConfigResponse = providers.PrepareProviderConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate()
	if len(diags) != 1 {
		t.Fatalf("wrong number of diagnostics %d; want %d", len(diags), 1)
	}
	if !strings.Contains(diags.Err().Error(), "bad") {
		t.Fatalf("bad: %s", diags.Err().Error())
	}
}

func TestContext2Validate_providerConfig_badEmpty(t *testing.T) {
	m := testModule(t, "validate-bad-pc-empty")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.PrepareProviderConfigResponse = providers.PrepareProviderConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_providerConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-pc")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_provisionerConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-prov-conf")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	pr := simpleMockProvisioner()

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	p.PrepareProviderConfigResponse = providers.PrepareProviderConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_provisionerConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-prov-conf")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	pr := simpleMockProvisioner()
	pr.ValidateProvisionerConfigFn = func(req provisioners.ValidateProvisionerConfigRequest) provisioners.ValidateProvisionerConfigResponse {
		var diags tfdiags.Diagnostics
		if req.Config.GetAttr("test_string").IsNull() {
			diags.Append(errors.New("test_string is not set"))
		}
		return provisioners.ValidateProvisionerConfigResponse{
			Diagnostics: diags,
		}
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_requiredVar(t *testing.T) {
	m := testModule(t, "validate-required-var")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"ami": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_resourceConfig_bad(t *testing.T) {
	m := testModule(t, "validate-bad-rc")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	p.ValidateResourceTypeConfigResponse = providers.ValidateResourceTypeConfigResponse{
		Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("bad")),
	}

	diags := c.Validate()
	if !diags.HasErrors() {
		t.Fatalf("succeeded; want error")
	}
}

func TestContext2Validate_resourceConfig_good(t *testing.T) {
	m := testModule(t, "validate-bad-rc")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_tainted(t *testing.T) {
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
					"num": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	m := testModule(t, "validate-good")
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:      "bar",
							Tainted: true,
						},
					},
				},
			},
		},
	})
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		State: state,
	})

	p.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		var diags tfdiags.Diagnostics
		if req.Config.GetAttr("foo").IsNull() {
			diags.Append(errors.New("foo is not set"))
		}
		return providers.ValidateResourceTypeConfigResponse{
			Diagnostics: diags,
		}
	}

	diags := c.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_targetedDestroy(t *testing.T) {
	m := testModule(t, "validate-targeted")
	p := testProvider("aws")
	pr := simpleMockProvisioner()
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
					"num": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Provisioners: map[string]ProvisionerFactory{
			"shell": testProvisionerFuncFixed(pr),
		},
		State: MustShimLegacyState(&State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"aws_instance.foo": resourceState("aws_instance", "i-bcd345"),
						"aws_instance.bar": resourceState("aws_instance", "i-abc123"),
					},
				},
			},
		}),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
		},
		Destroy: true,
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

func TestContext2Validate_varRefFilled(t *testing.T) {
	m := testModule(t, "validate-variable-ref")
	p := testProvider("aws")
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
	}
	c := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromCaller,
			},
		},
	})

	var value cty.Value
	p.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		value = req.Config.GetAttr("foo")
		return providers.ValidateResourceTypeConfigResponse{}
	}

	c.Validate()
	if !value.RawEquals(cty.StringVal("bar")) {
		t.Fatalf("bad: %#v", value)
	}
}

// Module variables weren't being interpolated during Validate phase.
// related to https://github.com/hashicorp/terraform/issues/5322
func TestContext2Validate_interpolateVar(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "input-interpolate-var")
	p := testProvider("null")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"template_file": {
				Attributes: map[string]*configschema.Attribute{
					"template": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"template": testProviderFuncFixed(p),
			},
		),
		UIInput: input,
	})

	diags := ctx.Validate()
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
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"attr": {Type: cty.String, Optional: true},
				},
			},
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		UIInput: input,
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

// Computed values are lost when a map is output from a module
func TestContext2Validate_interpolateMap(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "issue-9549")
	p := testProvider("template")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"template": testProviderFuncFixed(p),
			},
		),
		UIInput: input,
	})

	diags := ctx.Validate()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}

// Manually validate using the new PlanGraphBuilder
func TestContext2Validate_PlanGraphBuilder(t *testing.T) {
	fixture := contextFixtureApplyVars(t)
	opts := fixture.ContextOpts()
	opts.Variables = InputValues{
		"foo": &InputValue{
			Value:      cty.StringVal("us-east-1"),
			SourceType: ValueFromCaller,
		},
		"test_list": &InputValue{
			Value: cty.ListVal([]cty.Value{
				cty.StringVal("Hello"),
				cty.StringVal("World"),
			}),
			SourceType: ValueFromCaller,
		},
		"test_map": &InputValue{
			Value: cty.MapVal(map[string]cty.Value{
				"Hello": cty.StringVal("World"),
				"Foo":   cty.StringVal("Bar"),
				"Baz":   cty.StringVal("Foo"),
			}),
			SourceType: ValueFromCaller,
		},
		"amis": &InputValue{
			Value: cty.MapVal(map[string]cty.Value{
				"us-east-1": cty.StringVal("override"),
			}),
			SourceType: ValueFromCaller,
		},
	}
	c := testContext2(t, opts)

	graph, diags := (&PlanGraphBuilder{
		Config:     c.config,
		State:      states.NewState(),
		Components: c.components,
		Schemas:    c.schemas,
		Targets:    c.targets,
	}).Build(addrs.RootModuleInstance)
	if diags.HasErrors() {
		t.Fatalf("errors from PlanGraphBuilder: %s", diags.Err())
	}
	defer c.acquireRun("validate-test")()
	walker, diags := c.walk(graph, walkValidate)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	if len(walker.NonFatalDiagnostics) > 0 {
		t.Fatal(walker.NonFatalDiagnostics.Err())
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Unsupported attribute: This object does not have an attribute named "missing"
	if got, want := diags.Err().Error(), "Unsupported attribute"; strings.Index(got, want) == -1 {
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Unsupported attribute: This object does not have an attribute named "missing"
	if got, want := diags.Err().Error(), "Unsupported attribute"; strings.Index(got, want) == -1 {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Invalid resource count attribute: The special "count" attribute is no longer supported after Terraform v0.12. Instead, use length(aws_instance.test) to count resource instances.
	if got, want := diags.Err().Error(), "Invalid resource count attribute:"; strings.Index(got, want) == -1 {
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Reference to undeclared module: No module call named "foo" is declared in the root module.
	if got, want := diags.Err().Error(), "Reference to undeclared module:"; strings.Index(got, want) == -1 {
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Reference to undeclared module: No module call named "foo" is declared in the root module.
	if got, want := diags.Err().Error(), "Reference to undeclared module:"; strings.Index(got, want) == -1 {
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Reference to undeclared module: No module call named "foo" is declared in the root module.
	if got, want := diags.Err().Error(), "Reference to undeclared resource:"; strings.Index(got, want) == -1 {
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
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": testProviderFuncFixed(p),
			},
		),
	})

	diags := ctx.Validate()
	if !diags.HasErrors() {
		t.Fatal("succeeded; want errors")
	}
	// Should get this error:
	// Reference to undeclared module: No module call named "foo" is declared in the root module.
	if got, want := diags.Err().Error(), `no argument, nested block, or exported attribute named "does_not_exist_in_schema"`; strings.Index(got, want) == -1 {
		t.Fatalf("wrong error:\ngot:  %s\nwant: message containing %q", got, want)
	}
}
