package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestBuildProviderConfig(t *testing.T) {
	configBody := configs.SynthBody("", map[string]cty.Value{
		"set_in_config": cty.StringVal("config"),
	})
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
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
	rp := providers.Interface(provider)
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	n := &EvalConfigProvider{
		Addr:     providerAddr,
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

func TestEvalConfigProvider_unknownImport(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.UnknownVal(cty.String),
		}),
	}
	provider := mockProviderWithConfigSchema(simpleTestSchema())
	rp := providers.Interface(provider)
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	n := &EvalConfigProvider{
		Addr:                providerAddr,
		Config:              config,
		Provider:            &rp,
		VerifyConfigIsKnown: true,
	}

	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()

	_, err := n.Eval(ctx)

	var diags tfdiags.Diagnostics
	switch e := err.(type) {
	case tfdiags.NonFatalError:
		diags = e.Diagnostics
	default:
		t.Fatalf("expected err to be NonFatalError, was %T", err)
	}

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}

	if got, want := diags[0].Severity(), tfdiags.Error; got != want {
		t.Errorf("wrong diagnostic severity %#v; want %#v", got, want)
	}
	if got, want := diags[0].Description().Summary, "Invalid provider configuration"; got != want {
		t.Errorf("wrong diagnostic summary %#v; want %#v", got, want)
	}
	detail := `The configuration for provider["registry.terraform.io/hashicorp/foo"] depends on values that cannot be determined until apply.`
	if got, want := diags[0].Description().Detail, detail; got != want {
		t.Errorf("wrong diagnostic detail\n got: %q\nwant: %q", got, want)
	}

	if ctx.ConfigureProviderCalled {
		t.Fatal("should not be called")
	}
}

func TestEvalConfigProvider_unknownApply(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.UnknownVal(cty.String),
		}),
	}
	provider := mockProviderWithConfigSchema(simpleTestSchema())
	rp := providers.Interface(provider)
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	n := &EvalConfigProvider{
		Addr:                providerAddr,
		Config:              config,
		Provider:            &rp,
		VerifyConfigIsKnown: false,
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
	if got, want := gotObj.GetAttr("test_string"), cty.UnknownVal(cty.String); !got.RawEquals(want) {
		t.Errorf("wrong configuration value\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestEvalInitProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalInitProvider)
}

func TestEvalInitProvider(t *testing.T) {
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	n := &EvalInitProvider{
		Addr: providerAddr,
	}
	provider := &MockProvider{}
	ctx := &MockEvalContext{InitProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.InitProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.InitProviderAddr.String() != `provider["registry.terraform.io/hashicorp/foo"]` {
		t.Fatalf("wrong provider address %s", ctx.InitProviderAddr)
	}
}

func TestEvalCloseProvider(t *testing.T) {
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	n := &EvalCloseProvider{
		Addr: providerAddr,
	}
	provider := &MockProvider{}
	ctx := &MockEvalContext{CloseProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.CloseProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.CloseProviderAddr.String() != `provider["registry.terraform.io/hashicorp/foo"]` {
		t.Fatalf("wrong provider address %s", ctx.CloseProviderAddr)
	}
}

func TestEvalGetProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalGetProvider)
}

func TestEvalGetProvider(t *testing.T) {
	var actual providers.Interface
	n := &EvalGetProvider{
		Addr:   addrs.RootModuleInstance.ProviderConfigDefault(addrs.NewDefaultProvider("foo")),
		Output: &actual,
	}
	provider := &MockProvider{}
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
	if ctx.ProviderAddr.String() != `provider["registry.terraform.io/hashicorp/foo"]` {
		t.Fatalf("wrong provider address %s", ctx.ProviderAddr)
	}
}
