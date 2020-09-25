package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
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
