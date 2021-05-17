package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
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
