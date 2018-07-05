package terraform

import (
	"testing"

	"github.com/hashicorp/hcl2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/configs"
)

func TestBuildProviderConfig(t *testing.T) {
	configBody := configs.SynthBody("", map[string]cty.Value{
		"set_in_config": cty.StringVal("config"),
	})
	providerAddr := addrs.ProviderConfig{
		Type: "foo",
	}

	ctx := &MockEvalContext{
		// The input values map is expected to contain only keys that aren't
		// already present in the config, since we skip prompting for
		// attributes that are already set.
		ProviderInputValues: map[string]cty.Value{
			"set_by_input": cty.StringVal("input"),
		},
	}
	gotBody := buildProviderConfig(ctx, providerAddr, &configs.Provider{
		Name:   "foo",
		Config: configBody,
	})

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"set_in_config": {Type: cty.String, Optional: true},
			"set_by_input":  {Type: cty.String, Optional: true},
		},
	}
	got, diags := hcldec.Decode(gotBody, schema.DecoderSpec(), nil)
	if diags.HasErrors() {
		t.Fatalf("body decode failed: %s", diags.Error())
	}

	// We expect the provider config with the added input value
	want := cty.ObjectVal(map[string]cty.Value{
		"set_in_config": cty.StringVal("config"),
		"set_by_input":  cty.StringVal("input"),
	})
	if !got.RawEquals(want) {
		t.Fatalf("incorrect merged config\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestEvalConfigProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalConfigProvider)
}

func TestEvalConfigProvider(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("hello"),
		}),
	}
	provider := mockProviderWithConfigSchema(simpleTestSchema())
	rp := ResourceProvider(provider)
	n := &EvalConfigProvider{
		Addr:     addrs.ProviderConfig{Type: "foo"},
		Config:   config,
		Provider: &rp,
	}

	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.ConfigureProviderCalled {
		t.Fatal("should be called")
	}

	gotObj := ctx.ConfigureProviderConfig
	if !gotObj.Type().HasAttribute("test_string") {
		t.Fatal("configuration object does not have \"test_string\" attribute")
	}
	if got, want := gotObj.GetAttr("test_string"), cty.StringVal("hello"); !got.RawEquals(want) {
		t.Errorf("wrong configuration value\ngot:  %#v\nwant: %#v", got, want)
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
