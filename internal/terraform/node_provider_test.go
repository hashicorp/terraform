// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestNodeApplyableProviderExecute(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"user": cty.StringVal("hello"),
		}),
	}

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"user": {
				Type:     cty.String,
				Required: true,
			},
			"pw": {
				Type:     cty.String,
				Required: true,
			},
		},
	}
	provider := mockProviderWithConfigSchema(schema)
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}

	n := &NodeApplyableProvider{&NodeAbstractProvider{
		Addr:   providerAddr,
		Config: config,
	}}

	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()
	ctx.ProviderInputValues = map[string]cty.Value{
		"pw": cty.StringVal("so secret"),
	}

	if diags := n.Execute(ctx, walkApply); diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	if !ctx.ConfigureProviderCalled {
		t.Fatal("should be called")
	}

	gotObj := ctx.ConfigureProviderConfig
	if !gotObj.Type().HasAttribute("user") {
		t.Fatal("configuration object does not have \"user\" attribute")
	}
	if got, want := gotObj.GetAttr("user"), cty.StringVal("hello"); !got.RawEquals(want) {
		t.Errorf("wrong configuration value\ngot:  %#v\nwant: %#v", got, want)
	}

	if !gotObj.Type().HasAttribute("pw") {
		t.Fatal("configuration object does not have \"pw\" attribute")
	}
	if got, want := gotObj.GetAttr("pw"), cty.StringVal("so secret"); !got.RawEquals(want) {
		t.Errorf("wrong configuration value\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestNodeApplyableProviderExecute_unknownImport(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.UnknownVal(cty.String),
		}),
	}
	provider := mockProviderWithConfigSchema(simpleTestSchema())
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	n := &NodeApplyableProvider{&NodeAbstractProvider{
		Addr:   providerAddr,
		Config: config,
	}}

	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()

	diags := n.Execute(ctx, walkImport)
	if !diags.HasErrors() {
		t.Fatal("expected error, got success")
	}

	detail := `Invalid provider configuration: The configuration for provider["registry.terraform.io/hashicorp/foo"] depends on values that cannot be determined until apply.`
	if got, want := diags.Err().Error(), detail; got != want {
		t.Errorf("wrong diagnostic detail\n got: %q\nwant: %q", got, want)
	}

	if ctx.ConfigureProviderCalled {
		t.Fatal("should not be called")
	}
}

func TestNodeApplyableProviderExecute_unknownApply(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.UnknownVal(cty.String),
		}),
	}
	provider := mockProviderWithConfigSchema(simpleTestSchema())
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}
	n := &NodeApplyableProvider{&NodeAbstractProvider{
		Addr:   providerAddr,
		Config: config,
	}}
	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()

	if err := n.Execute(ctx, walkApply); err != nil {
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

func TestNodeApplyableProviderExecute_sensitive(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("hello").Mark(marks.Sensitive),
		}),
	}
	provider := mockProviderWithConfigSchema(simpleTestSchema())
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}

	n := &NodeApplyableProvider{&NodeAbstractProvider{
		Addr:   providerAddr,
		Config: config,
	}}

	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()
	if err := n.Execute(ctx, walkApply); err != nil {
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

func TestNodeApplyableProviderExecute_sensitiveValidate(t *testing.T) {
	config := &configs.Provider{
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("hello").Mark(marks.Sensitive),
		}),
	}
	provider := mockProviderWithConfigSchema(simpleTestSchema())
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}

	n := &NodeApplyableProvider{&NodeAbstractProvider{
		Addr:   providerAddr,
		Config: config,
	}}

	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()
	if err := n.Execute(ctx, walkValidate); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !provider.ValidateProviderConfigCalled {
		t.Fatal("should be called")
	}

	gotObj := provider.ValidateProviderConfigRequest.Config
	if !gotObj.Type().HasAttribute("test_string") {
		t.Fatal("configuration object does not have \"test_string\" attribute")
	}
	if got, want := gotObj.GetAttr("test_string"), cty.StringVal("hello"); !got.RawEquals(want) {
		t.Errorf("wrong configuration value\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestNodeApplyableProviderExecute_emptyValidate(t *testing.T) {
	config := &configs.Provider{
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
	}
	provider := mockProviderWithConfigSchema(&configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"test_string": {
				Type:     cty.String,
				Required: true,
			},
		},
	})
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("foo"),
	}

	n := &NodeApplyableProvider{&NodeAbstractProvider{
		Addr:   providerAddr,
		Config: config,
	}}

	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()
	if err := n.Execute(ctx, walkValidate); err != nil {
		t.Fatalf("err: %s", err)
	}

	if ctx.ConfigureProviderCalled {
		t.Fatal("should not be called")
	}
}

func TestNodeApplyableProvider_Validate(t *testing.T) {
	provider := mockProviderWithConfigSchema(&configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"region": {
				Type:     cty.String,
				Required: true,
			},
		},
	})
	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()

	t.Run("valid", func(t *testing.T) {
		config := &configs.Provider{
			Name: "test",
			Config: configs.SynthBody("", map[string]cty.Value{
				"region": cty.StringVal("mars"),
			}),
		}

		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr:   mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
				Config: config,
			},
		}

		diags := node.ValidateProvider(ctx, walkPlan, provider)
		if diags.HasErrors() {
			t.Errorf("unexpected error with valid config: %s", diags.Err())
		}
	})

	t.Run("invalid", func(t *testing.T) {
		config := &configs.Provider{
			Name: "test",
			Config: configs.SynthBody("", map[string]cty.Value{
				"region": cty.MapValEmpty(cty.String),
			}),
		}

		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr:   mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
				Config: config,
			},
		}

		diags := node.ValidateProvider(ctx, walkPlan, provider)
		if !diags.HasErrors() {
			t.Error("missing expected error with invalid config")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			},
		}

		diags := node.ValidateProvider(ctx, walkPlan, provider)
		if diags.HasErrors() {
			t.Errorf("unexpected error with empty config: %s", diags.Err())
		}
	})
}

// This test specifically tests responses from the
// providers.ValidateProviderConfigFn. See
// TestNodeApplyableProvider_ConfigProvider_config_fn_err for
// providers.ConfigureProviderRequest responses.
func TestNodeApplyableProvider_ConfigProvider(t *testing.T) {
	provider := mockProviderWithConfigSchema(&configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"region": {
				Type:     cty.String,
				Optional: true,
			},
		},
	})
	// For this test, we're returning an error for an optional argument. This
	// can happen for example if an argument is only conditionally required.
	provider.ValidateProviderConfigFn = func(req providers.ValidateProviderConfigRequest) (resp providers.ValidateProviderConfigResponse) {
		region := req.Config.GetAttr("region")
		if region.IsNull() {
			resp.Diagnostics = resp.Diagnostics.Append(
				tfdiags.WholeContainingBody(tfdiags.Error, "value is not found", "you did not supply a required value"))
		}
		return
	}

	// requiredProvider matches provider, but its attributes are required
	// explicitly. This means we can simulate an earlier failure in the
	// config validation.
	requiredProvider := mockProviderWithConfigSchema(&configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"region": {
				Type:     cty.String,
				Required: true,
			},
		},
	})

	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()

	t.Run("valid", func(t *testing.T) {
		config := &configs.Provider{
			Name: "test",
			Config: configs.SynthBody("", map[string]cty.Value{
				"region": cty.StringVal("mars"),
			}),
		}

		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr:   mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
				Config: config,
			},
		}

		diags := node.ConfigureProvider(ctx, walkPlan, provider, false)
		if diags.HasErrors() {
			t.Errorf("unexpected error with valid config: %s", diags.Err())
		}
	})

	t.Run("missing required config (no config at all)", func(t *testing.T) {
		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			},
		}

		diags := node.ConfigureProvider(ctx, walkPlan, provider, false)
		if !diags.HasErrors() {
			t.Fatal("missing expected error with nil config")
		}
		if !strings.Contains(diags.Err().Error(), "requires explicit configuration") {
			t.Errorf("diagnostic is missing \"requires explicit configuration\" message: %s", diags.Err())
		}
	})

	t.Run("missing required config", func(t *testing.T) {
		config := &configs.Provider{
			Name:   "test",
			Config: hcl.EmptyBody(),
		}
		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr:   mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
				Config: config,
			},
		}

		diags := node.ConfigureProvider(ctx, walkPlan, provider, false)
		if !diags.HasErrors() {
			t.Fatal("missing expected error with invalid config")
		}
		if !strings.Contains(diags.Err().Error(), "value is not found") {
			t.Errorf("wrong diagnostic: %s", diags.Err())
		}
	})

	t.Run("missing schema-required config (no config at all)", func(t *testing.T) {
		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			},
		}

		diags := node.ConfigureProvider(ctx, walkPlan, requiredProvider, false)
		if !diags.HasErrors() {
			t.Fatal("missing expected error with nil config")
		}
		if !strings.Contains(diags.Err().Error(), "requires explicit configuration") {
			t.Errorf("diagnostic is missing \"requires explicit configuration\" message: %s", diags.Err())
		}
	})

	t.Run("missing schema-required config", func(t *testing.T) {
		config := &configs.Provider{
			Name:   "test",
			Config: hcl.EmptyBody(),
		}
		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr:   mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
				Config: config,
			},
		}

		diags := node.ConfigureProvider(ctx, walkPlan, requiredProvider, false)
		if !diags.HasErrors() {
			t.Fatal("missing expected error with invalid config")
		}
		if !strings.Contains(diags.Err().Error(), "The argument \"region\" is required, but was not set.") {
			t.Errorf("wrong diagnostic: %s", diags.Err())
		}
	})

}

// This test is similar to TestNodeApplyableProvider_ConfigProvider, but tests responses from the providers.ConfigureProviderRequest
func TestNodeApplyableProvider_ConfigProvider_config_fn_err(t *testing.T) {
	provider := mockProviderWithConfigSchema(&configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"region": {
				Type:     cty.String,
				Optional: true,
			},
		},
	})
	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()
	// For this test, provider.PrepareConfigFn will succeed every time but the
	// ctx.ConfigureProviderFn will return an error if a value is not found.
	//
	// This is an unlikely but real situation that occurs:
	// https://github.com/hashicorp/terraform/issues/23087
	ctx.ConfigureProviderFn = func(addr addrs.AbsProviderConfig, cfg cty.Value) (diags tfdiags.Diagnostics) {
		if cfg.IsNull() {
			diags = diags.Append(fmt.Errorf("no config provided"))
		} else {
			region := cfg.GetAttr("region")
			if region.IsNull() {
				diags = diags.Append(fmt.Errorf("value is not found"))
			}
		}
		return
	}

	t.Run("valid", func(t *testing.T) {
		config := &configs.Provider{
			Name: "test",
			Config: configs.SynthBody("", map[string]cty.Value{
				"region": cty.StringVal("mars"),
			}),
		}

		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr:   mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
				Config: config,
			},
		}

		diags := node.ConfigureProvider(ctx, walkPlan, provider, false)
		if diags.HasErrors() {
			t.Errorf("unexpected error with valid config: %s", diags.Err())
		}
	})

	t.Run("missing required config (no config at all)", func(t *testing.T) {
		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			},
		}

		diags := node.ConfigureProvider(ctx, walkPlan, provider, false)
		if !diags.HasErrors() {
			t.Fatal("missing expected error with nil config")
		}
		if !strings.Contains(diags.Err().Error(), "requires explicit configuration") {
			t.Errorf("diagnostic is missing \"requires explicit configuration\" message: %s", diags.Err())
		}
	})

	t.Run("missing required config", func(t *testing.T) {
		config := &configs.Provider{
			Name:   "test",
			Config: hcl.EmptyBody(),
		}
		node := NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr:   mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
				Config: config,
			},
		}

		diags := node.ConfigureProvider(ctx, walkPlan, provider, false)
		if !diags.HasErrors() {
			t.Fatal("missing expected error with invalid config")
		}
		if diags.Err().Error() != "value is not found" {
			t.Errorf("wrong diagnostic: %s", diags.Err())
		}
	})
}

func TestGetSchemaError(t *testing.T) {
	provider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Diagnostics: tfdiags.Diagnostics.Append(nil, tfdiags.WholeContainingBody(tfdiags.Error, "oops", "error")),
		},
	}

	providerAddr := mustProviderConfig(`provider["terraform.io/some/provider"]`)
	ctx := &MockEvalContext{ProviderProvider: provider}
	ctx.installSimpleEval()
	node := NodeApplyableProvider{
		NodeAbstractProvider: &NodeAbstractProvider{
			Addr: providerAddr,
		},
	}

	diags := node.ConfigureProvider(ctx, walkPlan, provider, false)
	for _, d := range diags {
		desc := d.Description()
		if desc.Address != providerAddr.String() {
			t.Fatalf("missing provider address from diagnostics: %#v", desc)
		}
	}
}

func TestNodeApplyableProvider_providerVersion(t *testing.T) {
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("aws"),
	}

	node := &NodeApplyableProvider{
		NodeAbstractProvider: &NodeAbstractProvider{
			Addr: providerAddr,
		},
	}

	t.Run("with locked version", func(t *testing.T) {
		lock := depsfile.NewProviderLock(
			addrs.NewDefaultProvider("aws"),
			providerreqs.MustParseVersion("5.31.0"),
			nil,
			nil,
		)
		ctx := &MockEvalContext{
			ProviderLocksValue: map[addrs.Provider]*depsfile.ProviderLock{
				addrs.NewDefaultProvider("aws"): lock,
			},
		}

		got := node.providerVersion(ctx)
		if got != "5.31.0" {
			t.Errorf("wrong version\ngot:  %s\nwant: 5.31.0", got)
		}
	})

	t.Run("no lock file", func(t *testing.T) {
		ctx := &MockEvalContext{}

		got := node.providerVersion(ctx)
		if got != "" {
			t.Errorf("expected empty version, got: %s", got)
		}
	})

	t.Run("lock file without this provider", func(t *testing.T) {
		lock := depsfile.NewProviderLock(
			addrs.NewDefaultProvider("google"),
			providerreqs.MustParseVersion("4.0.0"),
			nil,
			nil,
		)
		ctx := &MockEvalContext{
			ProviderLocksValue: map[addrs.Provider]*depsfile.ProviderLock{
				addrs.NewDefaultProvider("google"): lock,
			},
		}

		got := node.providerVersion(ctx)
		if got != "" {
			t.Errorf("expected empty version, got: %s", got)
		}
	})
}

func TestNodeApplyableProvider_EvalPolicy_versionMeta(t *testing.T) {
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("aws"),
	}

	t.Run("version from lock file is passed in meta", func(t *testing.T) {
		lock := depsfile.NewProviderLock(
			addrs.NewDefaultProvider("aws"),
			providerreqs.MustParseVersion("5.31.0"),
			nil,
			nil,
		)

		mockPolicy := &policy.MockClient{}

		ctx := &MockEvalContext{
			ProviderLocksValue: map[addrs.Provider]*depsfile.ProviderLock{
				addrs.NewDefaultProvider("aws"): lock,
			},
			PolicyClientValue: mockPolicy,
		}

		node := &NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr: providerAddr,
			},
		}

		node.EvalPolicy(ctx, walkPlan, cty.EmptyObjectVal)

		if !mockPolicy.EvaluateProviderCalled {
			t.Fatal("EvaluateProvider was not called")
		}

		meta := mockPolicy.EvaluateProviderRequest.Meta

		gotVersion := meta.Version
		if gotVersion != "5.31.0" {
			t.Errorf("wrong version in meta\ngot:  %s\nwant: 5.31.0", gotVersion)
		}
	})

	t.Run("empty version when no lock file", func(t *testing.T) {
		mockPolicy := &policy.MockClient{}

		ctx := &MockEvalContext{
			PolicyClientValue: mockPolicy,
		}

		node := &NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr: providerAddr,
			},
		}

		node.EvalPolicy(ctx, walkPlan, cty.EmptyObjectVal)

		if !mockPolicy.EvaluateProviderCalled {
			t.Fatal("EvaluateProvider was not called")
		}

		meta := mockPolicy.EvaluateProviderRequest.Meta
		gotVersion := meta.Version
		if gotVersion != "" {
			t.Errorf("expected empty version in meta, got: %s", gotVersion)
		}
	})

	t.Run("no policy client skips evaluation", func(t *testing.T) {
		ctx := &MockEvalContext{}

		node := &NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr: providerAddr,
			},
		}

		diags := node.EvalPolicy(ctx, walkPlan, cty.EmptyObjectVal)
		if diags != nil {
			t.Fatalf("unexpected diagnostics: %s", diags.Err())
		}
	})

	t.Run("meta contains all expected fields", func(t *testing.T) {
		lock := depsfile.NewProviderLock(
			addrs.NewDefaultProvider("aws"),
			providerreqs.MustParseVersion("5.31.0"),
			nil,
			nil,
		)

		mockPolicy := &policy.MockClient{}

		ctx := &MockEvalContext{
			ProviderLocksValue: map[addrs.Provider]*depsfile.ProviderLock{
				addrs.NewDefaultProvider("aws"): lock,
			},
			PolicyClientValue: mockPolicy,
		}

		node := &NodeApplyableProvider{
			NodeAbstractProvider: &NodeAbstractProvider{
				Addr: providerAddr,
			},
		}

		node.EvalPolicy(ctx, walkPlan, cty.EmptyObjectVal)

		meta := mockPolicy.EvaluateProviderRequest.Meta

		checks := map[string]string{
			"name":        meta.Name,
			"alias":       meta.Alias,
			"type":        meta.Type,
			"namespace":   meta.Namespace,
			"source":      meta.Source,
			"module_path": meta.ModulePath,
			"version":     meta.Version,
		}
		expected := map[string]string{
			"name":        "aws",
			"alias":       "",
			"type":        "aws",
			"namespace":   "hashicorp",
			"source":      "registry.terraform.io/hashicorp/aws",
			"module_path": "",
			"version":     "5.31.0",
		}
		for field, got := range checks {
			want := expected[field]
			if got != want {
				t.Errorf("wrong meta %q\ngot:  %s\nwant: %s", field, got, want)
			}
		}
	})
}
