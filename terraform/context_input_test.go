package terraform

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

func TestContext2Input_provider(t *testing.T) {
	m := testModule(t, "input-provider")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {
					Type:        cty.String,
					Required:    true,
					Description: "something something",
				},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {},
		},
	}

	inp := &MockUIInput{
		InputReturnMap: map[string]string{
			"provider.aws.foo": "bar",
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		UIInput: inp,
	})

	var actual interface{}
	p.ConfigureFn = func(c *ResourceConfig) error {
		actual = c.Config["foo"]
		return nil
	}
	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		return nil, c.CheckSet([]string{"foo"})
	}

	if diags := ctx.Input(InputModeStd); diags.HasErrors() {
		t.Fatalf("input errors: %s", diags.Err())
	}

	if !inp.InputCalled {
		t.Fatal("no input prompt; want prompt for argument \"foo\"")
	}
	if got, want := inp.InputOpts.Description, "something something"; got != want {
		t.Errorf("wrong description\ngot:  %q\nwant: %q", got, want)
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	if !reflect.DeepEqual(actual, "bar") {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", actual, "bar")
	}
}

func TestContext2Input_providerMulti(t *testing.T) {
	m := testModule(t, "input-provider-multi")

	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {
					Type:        cty.String,
					Required:    true,
					Description: "something something",
				},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {},
		},
	}

	inp := &MockUIInput{
		InputReturnMap: map[string]string{
			"provider.aws.foo":      "bar",
			"provider.aws.east.foo": "bar",
		},
	}

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		UIInput: inp,
	})

	var actual []interface{}
	var lock sync.Mutex
	p.ValidateFn = func(c *ResourceConfig) ([]string, []error) {
		return nil, c.CheckSet([]string{"foo"})
	}

	if diags := ctx.Input(InputModeStd); diags.HasErrors() {
		t.Fatalf("input errors: %s", diags.Err())
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	p.ConfigureFn = func(c *ResourceConfig) error {
		lock.Lock()
		defer lock.Unlock()
		actual = append(actual, c.Config["foo"])
		return nil
	}
	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	expected := []interface{}{"bar", "bar"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", actual, expected)
	}
}

func TestContext2Input_providerOnce(t *testing.T) {
	m := testModule(t, "input-provider-once")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
	})

	//count := 0
	/*p.InputFn = func(i UIInput, c *ResourceConfig) (*ResourceConfig, error) {
		count++
		_, set := c.Config["from_input"]

		if count == 1 {
			if set {
				return nil, errors.New("from_input should not be set")
			}
			c.Config["from_input"] = "x"
		}

		if count > 1 && !set {
			return nil, errors.New("from_input should be set")
		}

		return c, nil
	}*/

	if diags := ctx.Input(InputModeStd); diags.HasErrors() {
		t.Fatalf("input errors: %s", diags.Err())
	}
}

func TestContext2Input_providerId(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "input-provider")

	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {
					Type:        cty.String,
					Required:    true,
					Description: "something something",
				},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {},
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

	var actual interface{}
	p.ConfigureFn = func(c *ResourceConfig) error {
		actual = c.Config["foo"]
		return nil
	}

	input.InputReturnMap = map[string]string{
		"provider.aws.foo": "bar",
	}

	if diags := ctx.Input(InputModeStd); diags.HasErrors() {
		t.Fatalf("input errors: %s", diags.Err())
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	if !reflect.DeepEqual(actual, "bar") {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", actual, "bar")
	}
}

func TestContext2Input_providerOnly(t *testing.T) {
	input := new(MockUIInput)

	m := testModule(t, "input-provider-vars")

	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	p.GetSchemaReturn = &ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {
					Type:     cty.String,
					Required: true,
				},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"foo":  {Type: cty.String, Required: true},
					"id":   {Type: cty.String, Computed: true},
					"type": {Type: cty.String, Computed: true},
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
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("us-west-2"),
				SourceType: ValueFromCaller,
			},
		},
		UIInput: input,
	})

	input.InputReturnMap = map[string]string{
		"provider.aws.foo": "bar",
	}

	var actual interface{}
	p.ConfigureFn = func(c *ResourceConfig) error {
		actual = c.Config["foo"]
		return nil
	}

	if err := ctx.Input(InputModeProvider); err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	state, err := ctx.Apply()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, "bar") {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", actual, "bar")
	}

	actualStr := strings.TrimSpace(state.String())
	expectedStr := strings.TrimSpace(testTerraformInputProviderOnlyStr)
	if actualStr != expectedStr {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actualStr, expectedStr)
	}
}

func TestContext2Input_providerVars(t *testing.T) {
	input := new(MockUIInput)
	m := testModule(t, "input-provider-with-vars")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
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
		UIInput: input,
	})

	input.InputReturnMap = map[string]string{
		"var.foo": "bar",
	}

	var actual interface{}
	/*p.InputFn = func(i UIInput, c *ResourceConfig) (*ResourceConfig, error) {
		c.Config["bar"] = "baz"
		return c, nil
	}*/
	p.ConfigureFn = func(c *ResourceConfig) error {
		actual, _ = c.Get("foo")
		return nil
	}

	if diags := ctx.Input(InputModeStd); diags.HasErrors() {
		t.Fatalf("input errors: %s", diags.Err())
	}

	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}

	if _, diags := ctx.Apply(); diags.HasErrors() {
		t.Fatalf("apply errors: %s", diags.Err())
	}

	if !reflect.DeepEqual(actual, "bar") {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestContext2Input_providerVarsModuleInherit(t *testing.T) {
	input := new(MockUIInput)
	m := testModule(t, "input-provider-with-vars-and-module")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		UIInput: input,
	})

	/*p.InputFn = func(i UIInput, c *ResourceConfig) (*ResourceConfig, error) {
		if errs := c.CheckSet([]string{"access_key"}); len(errs) > 0 {
			return c, errs[0]
		}
		return c, nil
	}*/
	p.ConfigureFn = func(c *ResourceConfig) error {
		return nil
	}

	if diags := ctx.Input(InputModeStd); diags.HasErrors() {
		t.Fatalf("input errors: %s", diags.Err())
	}
}

// adding a list interpolation in fails to interpolate the count variable
func TestContext2Input_submoduleTriggersInvalidCount(t *testing.T) {
	input := new(MockUIInput)
	m := testModule(t, "input-submodule-count")
	p := testProvider("aws")
	p.ApplyFn = testApplyFn
	p.DiffFn = testDiffFn
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"aws": testProviderFuncFixed(p),
			},
		),
		UIInput: input,
	})

	/*p.InputFn = func(i UIInput, c *ResourceConfig) (*ResourceConfig, error) {
		return c, nil
	}*/
	p.ConfigureFn = func(c *ResourceConfig) error {
		return nil
	}

	if diags := ctx.Input(InputModeStd); diags.HasErrors() {
		t.Fatalf("input errors: %s", diags.Err())
	}
}

// In this case, a module variable can't be resolved from a data source until
// it's refreshed, but it can't be refreshed during Input.
func TestContext2Input_dataSourceRequiresRefresh(t *testing.T) {
	input := new(MockUIInput)
	p := testProvider("null")
	m := testModule(t, "input-module-data-vars")

	p.GetSchemaReturn = &ProviderSchema{
		DataSources: map[string]*configschema.Block{
			"null_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.List(cty.String), Optional: true},
				},
			},
		},
	}
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		return providers.ReadDataSourceResponse{
			State: req.Config,
		}
	}

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "null_data_source",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsFlat: map[string]string{
					"id":    "-",
					"foo.#": "1",
					"foo.0": "a",
					// foo.1 exists in the data source, but needs to be refreshed.
				},
				Status: states.ObjectReady,
			},
			addrs.ProviderConfig{Type: "null"}.Absolute(addrs.RootModuleInstance),
		)
	})

	ctx := testContext2(t, &ContextOpts{
		Config: m,
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"null": testProviderFuncFixed(p),
			},
		),
		State:   state,
		UIInput: input,
	})

	if diags := ctx.Input(InputModeStd); diags.HasErrors() {
		t.Fatalf("input errors: %s", diags.Err())
	}

	// ensure that plan works after Refresh
	if _, diags := ctx.Refresh(); diags.HasErrors() {
		t.Fatalf("refresh errors: %s", diags.Err())
	}
	if _, diags := ctx.Plan(); diags.HasErrors() {
		t.Fatalf("plan errors: %s", diags.Err())
	}
}
