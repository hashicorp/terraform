package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestEvalBuildProviderConfig_impl(t *testing.T) {
	var _ EvalNode = new(EvalBuildProviderConfig)
}

func TestEvalBuildProviderConfig(t *testing.T) {
	config := testResourceConfig(t, map[string]interface{}{
		"set_in_config":            "config",
		"set_in_config_and_parent": "config",
		"computed_in_config":       "config",
	})
	provider := "foo"

	n := &EvalBuildProviderConfig{
		Provider: provider,
		Config:   &config,
		Output:   &config,
	}

	ctx := &MockEvalContext{
		ParentProviderConfigConfig: testResourceConfig(t, map[string]interface{}{
			"inherited_from_parent":    "parent",
			"set_in_config_and_parent": "parent",
		}),
		ProviderInputConfig: map[string]interface{}{
			"set_in_config": "input",
			"set_by_input":  "input",
		},
	}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	// This is a merger of the following, with later items taking precedence:
	// - "config" (the config as written in the current module, with all
	//   interpolation expressions resolved)
	// - ProviderInputConfig (mock of config produced by the input walk, after
	//   prompting the user interactively for values unspecified in config)
	// - ParentProviderConfigConfig (mock of config inherited from a parent module)
	expected := map[string]interface{}{
		"set_in_config":            "input", // in practice, input map contains identical literals from config
		"set_in_config_and_parent": "parent",
		"inherited_from_parent":    "parent",
		"computed_in_config":       "config",
		"set_by_input":             "input",
	}
	if !reflect.DeepEqual(config.Raw, expected) {
		t.Fatalf("incorrect merged config %#v; want %#v", config.Raw, expected)
	}
}

func TestEvalBuildProviderConfig_parentPriority(t *testing.T) {
	config := testResourceConfig(t, map[string]interface{}{})
	provider := "foo"

	n := &EvalBuildProviderConfig{
		Provider: provider,
		Config:   &config,
		Output:   &config,
	}

	ctx := &MockEvalContext{
		ParentProviderConfigConfig: testResourceConfig(t, map[string]interface{}{
			"foo": "bar",
		}),
		ProviderInputConfig: map[string]interface{}{
			"foo": "baz",
		},
	}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]interface{}{
		"foo": "bar",
	}
	if !reflect.DeepEqual(config.Raw, expected) {
		t.Fatalf("bad: %#v", config.Raw)
	}
}

func TestEvalConfigProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalConfigProvider)
}

func TestEvalConfigProvider(t *testing.T) {
	config := testResourceConfig(t, map[string]interface{}{})
	provider := &MockResourceProvider{}
	n := &EvalConfigProvider{Config: &config}

	ctx := &MockEvalContext{ProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.ConfigureProviderCalled {
		t.Fatal("should be called")
	}
	if !reflect.DeepEqual(ctx.ConfigureProviderConfig, config) {
		t.Fatalf("bad: %#v", ctx.ConfigureProviderConfig)
	}
}

func TestEvalInitProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalInitProvider)
}

func TestEvalInitProvider(t *testing.T) {
	n := &EvalInitProvider{Name: "foo"}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{InitProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.InitProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.InitProviderName != "foo" {
		t.Fatalf("bad: %#v", ctx.InitProviderName)
	}
}

func TestEvalCloseProvider(t *testing.T) {
	n := &EvalCloseProvider{Name: "foo"}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{CloseProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.CloseProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.CloseProviderName != "foo" {
		t.Fatalf("bad: %#v", ctx.CloseProviderName)
	}
}

func TestEvalGetProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalGetProvider)
}

func TestEvalGetProvider(t *testing.T) {
	var actual ResourceProvider
	n := &EvalGetProvider{Name: "foo", Output: &actual}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{ProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != provider {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.ProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.ProviderName != "foo" {
		t.Fatalf("bad: %#v", ctx.ProviderName)
	}
}

func TestEvalInputProvider(t *testing.T) {
	var provider ResourceProvider = &MockResourceProvider{
		InputFn: func(ui UIInput, c *ResourceConfig) (*ResourceConfig, error) {
			if c.Config["mock_config"] != "mock" {
				t.Fatalf("original config not passed to provider.Input")
			}

			rawConfig, err := config.NewRawConfig(map[string]interface{}{
				"set_in_config": "input",
				"set_by_input":  "input",
				"computed":      "fake_computed",
			})
			if err != nil {
				return nil, err
			}
			config := NewResourceConfig(rawConfig)
			config.ComputedKeys = []string{"computed"} // fake computed key

			return config, nil
		},
	}
	ctx := &MockEvalContext{ProviderProvider: provider}
	rawConfig, err := config.NewRawConfig(map[string]interface{}{
		"mock_config": "mock",
	})
	if err != nil {
		t.Fatalf("NewRawConfig failed: %s", err)
	}
	config := NewResourceConfig(rawConfig)

	n := &EvalInputProvider{
		Name:     "mock",
		Provider: &provider,
		Config:   &config,
	}

	result, err := n.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval failed: %s", err)
	}
	if result != nil {
		t.Fatalf("Eval returned non-nil result %#v", result)
	}

	if !ctx.SetProviderInputCalled {
		t.Fatalf("ctx.SetProviderInput wasn't called")
	}

	if got, want := ctx.SetProviderInputName, "mock"; got != want {
		t.Errorf("wrong provider name %q; want %q", got, want)
	}

	inputCfg := ctx.SetProviderInputConfig
	want := map[string]interface{}{
		"set_in_config": "input",
		"set_by_input":  "input",
		// "computed" is omitted because it value isn't known at input time
	}
	if !reflect.DeepEqual(inputCfg, want) {
		t.Errorf("got incorrect input config %#v; want %#v", inputCfg, want)
	}
}
