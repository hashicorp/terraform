package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"
)

// contextTestFixture is a container for a set of objects that work together
// to create a base testing scenario. This is used to represent some common
// situations used as the basis for multiple tests.
type contextTestFixture struct {
	Config           *configs.Config
	ProviderResolver ResourceProviderResolver
	Provisioners     map[string]ResourceProvisionerFactory
}

// ContextOpts returns a ContextOps pre-populated with the elements of this
// fixture. Each call returns a distinct object, so callers can apply further
// _shallow_ modifications to the options as needed.
func (f *contextTestFixture) ContextOpts() *ContextOpts {
	return &ContextOpts{
		Config:           f.Config,
		ProviderResolver: f.ProviderResolver,
		Provisioners:     f.Provisioners,
	}
}

// contextFixtureApplyVars builds and returns a test fixture for testing
// input variables, primarily during the apply phase. The configuration is
// loaded from test-fixtures/apply-vars, and the provider resolver is
// configured with a resource type schema for aws_instance that matches
// what's used in that configuration.
func contextFixtureApplyVars(t *testing.T) *contextTestFixture {
	c := testModule(t, "apply-vars")
	p := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo":  {Type: cty.String, Optional: true},
			"bar":  {Type: cty.String, Optional: true},
			"baz":  {Type: cty.String, Optional: true},
			"num":  {Type: cty.Number, Optional: true},
			"list": {Type: cty.List(cty.String), Optional: true},
			"map":  {Type: cty.Map(cty.String), Optional: true},
		},
	})
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	return &contextTestFixture{
		Config: c,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	}
}

// contextFixtureApplyVarsEnv builds and returns a test fixture for testing
// input variables set from the environment. The configuration is
// loaded from test-fixtures/apply-vars-env, and the provider resolver is
// configured with a resource type schema for aws_instance that matches
// what's used in that configuration.
func contextFixtureApplyVarsEnv(t *testing.T) *contextTestFixture {
	c := testModule(t, "apply-vars-env")
	p := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"string": {Type: cty.String, Optional: true},
			"list":   {Type: cty.List(cty.String), Optional: true},
			"map":    {Type: cty.Map(cty.String), Optional: true},
		},
	})
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	return &contextTestFixture{
		Config: c,
		ProviderResolver: ResourceProviderResolverFixed(
			map[string]ResourceProviderFactory{
				"aws": testProviderFuncFixed(p),
			},
		),
	}
}
