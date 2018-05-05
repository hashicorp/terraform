package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/config"
)

func TestBuildProviderConfig(t *testing.T) {
	configBody := configs.SynthBody("", map[string]cty.Value{
		"set_in_config":            cty.StringVal("config"),
		"set_in_config_and_parent": cty.StringVal("config"),
		"computed_in_config":       cty.StringVal("config"),
	})
	providerAddr := addrs.ProviderConfig{
		Type: "foo",
	}

	ctx := &MockEvalContext{
		ProviderInputValues: map[string]cty.Value{
			"set_in_config": cty.StringVal("input"),
			"set_by_input":  cty.StringVal("input"),
		},
	}
	got := buildProviderConfig(ctx, providerAddr, configBody)

	// We expect the provider config with the added input value
	want := map[string]cty.Value{
		"set_in_config":            cty.StringVal("config"),
		"set_in_config_and_parent": cty.StringVal("config"),
		"computed_in_config":       cty.StringVal("config"),
		"set_by_input":             cty.StringVal("input"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("incorrect merged config\ngot:  %#v\nwant: \n%#v", got, want)
	}
}

func TestEvalConfigProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalConfigProvider)
}

func TestEvalConfigProvider(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
	}
	provider := &MockResourceProvider{}
	n := &EvalConfigProvider{Config: config}

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
	n := &EvalInitProvider{
		Addr: addrs.ProviderConfig{Type: "foo"},
	}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{InitProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.InitProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.InitProviderAddr.String() != "provider.foo" {
		t.Fatalf("wrong provider address %s", ctx.InitProviderAddr)
	}
}

func TestEvalCloseProvider(t *testing.T) {
	n := &EvalCloseProvider{
		Addr: addrs.ProviderConfig{Type: "foo"},
	}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{CloseProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.CloseProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.CloseProviderAddr.String() != "provider.foo" {
		t.Fatalf("wrong provider address %s", ctx.CloseProviderAddr)
	}
}

func TestEvalGetProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalGetProvider)
}

func TestEvalGetProvider(t *testing.T) {
	var actual ResourceProvider
	n := &EvalGetProvider{
		Addr:   addrs.RootModuleInstance.ProviderConfigDefault("foo"),
		Output: &actual,
	}
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
	if ctx.ProviderAddr.String() != "provider.foo" {
		t.Fatalf("wrong provider address %s", ctx.ProviderAddr)
	}
}

func TestEvalInputProvider(t *testing.T) {
	var provider ResourceProvider = &MockResourceProvider{
		InputFn: func(ui UIInput, c *ResourceConfig) (*ResourceConfig, error) {
			if c.Config["mock_config"] != "mock" {
				t.Fatalf("original config not passed to provider.Input")
			}

			rawConfig, err := config.NewRawConfig(map[string]interface{}{
				"set_by_input": "input",
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
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("synth", map[string]cty.Value{
			"mock_config":   cty.StringVal("mock"),
			"set_in_config": cty.StringVal("input"),
		}),
	}

	n := &EvalInputProvider{
		Addr:     addrs.ProviderConfig{Type: "foo"},
		Provider: &provider,
		Config:   config,
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

	if got, want := ctx.SetProviderInputAddr.String(), "provider.mock"; got != want {
		t.Errorf("wrong provider name %q; want %q", got, want)
	}

	inputCfg := ctx.SetProviderInputValues

	// we should only have the value that was set during Input
	want := map[string]cty.Value{
		"set_by_input": cty.StringVal("input"),
	}
	if !reflect.DeepEqual(inputCfg, want) {
		t.Errorf("got incorrect input config:\n%#v\nwant:\n%#v", inputCfg, want)
	}
}
