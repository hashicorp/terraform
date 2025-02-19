// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestNodeValidatableResource_ValidateProvisioner_valid(t *testing.T) {
	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	mp := &MockProvisioner{}
	ps := &configschema.Block{}
	ctx.ProvisionerSchemaSchema = ps
	ctx.ProvisionerProvisioner = mp

	pc := &configs.Provisioner{
		Type:   "baz",
		Config: hcl.EmptyBody(),
		Connection: &configs.Connection{
			Config: configs.SynthBody("", map[string]cty.Value{
				"host": cty.StringVal("localhost"),
				"type": cty.StringVal("ssh"),
				"port": cty.NumberIntVal(10022),
			}),
		},
	}

	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "test_foo",
		Name:   "bar",
		Config: configs.SynthBody("", map[string]cty.Value{}),
	}

	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:   mustConfigResourceAddr("test_foo.bar"),
			Config: rc,
		},
	}

	diags := node.validateProvisioner(ctx, pc, nil)
	if diags.HasErrors() {
		t.Fatalf("node.Eval failed: %s", diags.Err())
	}
	if !mp.ValidateProvisionerConfigCalled {
		t.Fatalf("p.ValidateProvisionerConfig not called")
	}
}

func TestNodeValidatableResource_ValidateProvisioner__warning(t *testing.T) {
	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	mp := &MockProvisioner{}
	ps := &configschema.Block{}
	ctx.ProvisionerSchemaSchema = ps
	ctx.ProvisionerProvisioner = mp

	pc := &configs.Provisioner{
		Type:   "baz",
		Config: hcl.EmptyBody(),
	}

	rc := &configs.Resource{
		Mode:    addrs.ManagedResourceMode,
		Type:    "test_foo",
		Name:    "bar",
		Config:  configs.SynthBody("", map[string]cty.Value{}),
		Managed: &configs.ManagedResource{},
	}

	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:   mustConfigResourceAddr("test_foo.bar"),
			Config: rc,
		},
	}

	{
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.SimpleWarning("foo is deprecated"))
		mp.ValidateProvisionerConfigResponse = provisioners.ValidateProvisionerConfigResponse{
			Diagnostics: diags,
		}
	}

	diags := node.validateProvisioner(ctx, pc, nil)
	if len(diags) != 1 {
		t.Fatalf("wrong number of diagnostics in %s; want one warning", diags.ErrWithWarnings())
	}

	if got, want := diags[0].Description().Summary, mp.ValidateProvisionerConfigResponse.Diagnostics[0].Description().Summary; got != want {
		t.Fatalf("wrong warning %q; want %q", got, want)
	}
}

func TestNodeValidatableResource_ValidateProvisioner__connectionInvalid(t *testing.T) {
	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	mp := &MockProvisioner{}
	ps := &configschema.Block{}
	ctx.ProvisionerSchemaSchema = ps
	ctx.ProvisionerProvisioner = mp

	pc := &configs.Provisioner{
		Type:   "baz",
		Config: hcl.EmptyBody(),
		Connection: &configs.Connection{
			Config: configs.SynthBody("", map[string]cty.Value{
				"type":             cty.StringVal("ssh"),
				"bananananananana": cty.StringVal("foo"),
				"bazaz":            cty.StringVal("bar"),
			}),
		},
	}

	rc := &configs.Resource{
		Mode:    addrs.ManagedResourceMode,
		Type:    "test_foo",
		Name:    "bar",
		Config:  configs.SynthBody("", map[string]cty.Value{}),
		Managed: &configs.ManagedResource{},
	}

	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:   mustConfigResourceAddr("test_foo.bar"),
			Config: rc,
		},
	}

	diags := node.validateProvisioner(ctx, pc, nil)
	if !diags.HasErrors() {
		t.Fatalf("node.Eval succeeded; want error")
	}
	if len(diags) != 3 {
		t.Fatalf("wrong number of diagnostics; want two errors\n\n%s", diags.Err())
	}

	errStr := diags.Err().Error()
	if !(strings.Contains(errStr, "bananananananana") && strings.Contains(errStr, "bazaz")) {
		t.Fatalf("wrong errors %q; want something about each of our invalid connInfo keys", errStr)
	}
}

func TestNodeValidatableResource_ValidateProvisioner_baseConnInvalid(t *testing.T) {
	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	mp := &MockProvisioner{}
	ps := &configschema.Block{}
	ctx.ProvisionerSchemaSchema = ps
	ctx.ProvisionerProvisioner = mp

	pc := &configs.Provisioner{
		Type:   "baz",
		Config: hcl.EmptyBody(),
	}

	baseConn := &configs.Connection{
		Config: configs.SynthBody("", map[string]cty.Value{
			"type":             cty.StringVal("ssh"),
			"bananananananana": cty.StringVal("foo"),
			"bazaz":            cty.StringVal("bar"),
		}),
	}

	rc := &configs.Resource{
		Mode:    addrs.ManagedResourceMode,
		Type:    "test_foo",
		Name:    "bar",
		Config:  configs.SynthBody("", map[string]cty.Value{}),
		Managed: &configs.ManagedResource{},
	}

	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:   mustConfigResourceAddr("test_foo.bar"),
			Config: rc,
		},
	}

	diags := node.validateProvisioner(ctx, pc, baseConn)
	if !diags.HasErrors() {
		t.Fatalf("node.Eval succeeded; want error")
	}
	if len(diags) != 3 {
		t.Fatalf("wrong number of diagnostics; want two errors\n\n%s", diags.Err())
	}

	errStr := diags.Err().Error()
	if !(strings.Contains(errStr, "bananananananana") && strings.Contains(errStr, "bazaz")) {
		t.Fatalf("wrong errors %q; want something about each of our invalid connInfo keys", errStr)
	}
}

func TestNodeValidatableResource_ValidateResource_managedResource(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		if got, want := req.TypeName, "test_object"; got != want {
			t.Fatalf("wrong resource type\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_string"), cty.StringVal("bar"); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_string\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_number"), cty.NumberIntVal(2); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_number\ngot:  %#v\nwant: %#v", got, want)
		}
		return providers.ValidateResourceConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_object",
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("bar"),
			"test_number": cty.NumberIntVal(2).Mark(marks.Sensitive),
		}),
	}
	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:             mustConfigResourceAddr("test_foo.bar"),
			Config:           rc,
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	ctx.ProviderSchemaSchema = mp.GetProviderSchema()
	ctx.ProviderProvider = p

	err := node.validateResource(ctx)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mp.ValidateResourceConfigCalled {
		t.Fatal("Expected ValidateResourceConfig to be called, but it was not!")
	}
}

func TestNodeValidatableResource_ValidateResource_managedResourceCount(t *testing.T) {
	// Setup
	mp := simpleMockProvider()
	mp.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		if got, want := req.TypeName, "test_object"; got != want {
			t.Fatalf("wrong resource type\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_string"), cty.StringVal("bar"); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_string\ngot:  %#v\nwant: %#v", got, want)
		}
		return providers.ValidateResourceConfigResponse{}
	}

	p := providers.Interface(mp)

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	ctx.ProviderSchemaSchema = mp.GetProviderSchema()
	ctx.ProviderProvider = p

	tests := []struct {
		name  string
		count hcl.Expression
	}{
		{
			"simple count",
			hcltest.MockExprLiteral(cty.NumberIntVal(2)),
		},
		{
			"marked count value",
			hcltest.MockExprLiteral(cty.NumberIntVal(3).Mark("marked")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rc := &configs.Resource{
				Mode:  addrs.ManagedResourceMode,
				Type:  "test_object",
				Name:  "foo",
				Count: test.count,
				Config: configs.SynthBody("", map[string]cty.Value{
					"test_string": cty.StringVal("bar"),
				}),
			}
			node := NodeValidatableResource{
				NodeAbstractResource: &NodeAbstractResource{
					Addr:             mustConfigResourceAddr("test_foo.bar"),
					Config:           rc,
					ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
				},
			}

			diags := node.validateResource(ctx)
			if diags.HasErrors() {
				t.Fatalf("err: %s", diags.Err())
			}

			if !mp.ValidateResourceConfigCalled {
				t.Fatal("Expected ValidateResourceConfig to be called, but it was not!")
			}
		})
	}
}

func TestNodeValidatableResource_ValidateResource_dataSource(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateDataResourceConfigFn = func(req providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
		if got, want := req.TypeName, "test_object"; got != want {
			t.Fatalf("wrong resource type\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_string"), cty.StringVal("bar"); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_string\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_number"), cty.NumberIntVal(2); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_number\ngot:  %#v\nwant: %#v", got, want)
		}
		return providers.ValidateDataResourceConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "test_object",
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("bar"),
			"test_number": cty.NumberIntVal(2).Mark(marks.Sensitive),
		}),
	}

	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:             mustConfigResourceAddr("test_foo.bar"),
			Config:           rc,
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	ctx.ProviderSchemaSchema = mp.GetProviderSchema()
	ctx.ProviderProvider = p

	diags := node.validateResource(ctx)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}

	if !mp.ValidateDataResourceConfigCalled {
		t.Fatal("Expected ValidateDataSourceConfig to be called, but it was not!")
	}
}

func TestNodeValidatableResource_ValidateResource_valid(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		return providers.ValidateResourceConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "test_object",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
	}
	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:             mustConfigResourceAddr("test_object.foo"),
			Config:           rc,
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	ctx.ProviderSchemaSchema = mp.GetProviderSchema()
	ctx.ProviderProvider = p

	diags := node.validateResource(ctx)
	if diags.HasErrors() {
		t.Fatalf("err: %s", diags.Err())
	}
}

func TestNodeValidatableResource_ValidateResource_warningsAndErrorsPassedThrough(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.SimpleWarning("warn"))
		diags = diags.Append(errors.New("err"))
		return providers.ValidateResourceConfigResponse{
			Diagnostics: diags,
		}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "test_object",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
	}
	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:             mustConfigResourceAddr("test_foo.bar"),
			Config:           rc,
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()
	ctx.ProviderSchemaSchema = mp.GetProviderSchema()
	ctx.ProviderProvider = p

	diags := node.validateResource(ctx)
	if !diags.HasErrors() {
		t.Fatal("unexpected success; want error")
	}

	bySeverity := map[tfdiags.Severity]tfdiags.Diagnostics{}
	for _, diag := range diags {
		bySeverity[diag.Severity()] = append(bySeverity[diag.Severity()], diag)
	}
	if len(bySeverity[tfdiags.Warning]) != 1 || bySeverity[tfdiags.Warning][0].Description().Summary != "warn" {
		t.Errorf("Expected 1 warning 'warn', got: %s", bySeverity[tfdiags.Warning].ErrWithWarnings())
	}
	if len(bySeverity[tfdiags.Error]) != 1 || bySeverity[tfdiags.Error][0].Description().Summary != "err" {
		t.Errorf("Expected 1 error 'err', got: %s", bySeverity[tfdiags.Error].Err())
	}
}

func TestNodeValidatableResource_ValidateResource_invalidDependsOn(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		return providers.ValidateResourceConfigResponse{}
	}

	// We'll check a _valid_ config first, to make sure we're not failing
	// for some other reason, and then make it invalid.
	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "test_object",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
		DependsOn: []hcl.Traversal{
			// Depending on path.module is pointless, since it is immediately
			// available, but we allow all of the referencable addrs here
			// for consistency: referencing them is harmless, and avoids the
			// need for us to document a different subset of addresses that
			// are valid in depends_on.
			// For the sake of this test, it's a valid address we can use that
			// doesn't require something else to exist in the configuration.
			{
				hcl.TraverseRoot{
					Name: "path",
				},
				hcl.TraverseAttr{
					Name: "module",
				},
			},
		},
	}
	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:             mustConfigResourceAddr("test_foo.bar"),
			Config:           rc,
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	ctx.ProviderSchemaSchema = mp.GetProviderSchema()
	ctx.ProviderProvider = p

	diags := node.validateResource(ctx)
	if diags.HasErrors() {
		t.Fatalf("error for supposedly-valid config: %s", diags.ErrWithWarnings())
	}

	// Now we'll make it invalid by adding additional traversal steps at
	// the end of what we're referencing. This is intended to catch the
	// situation where the user tries to depend on e.g. a specific resource
	// attribute, rather than the whole resource, like aws_instance.foo.id.
	rc.DependsOn = append(rc.DependsOn, hcl.Traversal{
		hcl.TraverseRoot{
			Name: "path",
		},
		hcl.TraverseAttr{
			Name: "module",
		},
		hcl.TraverseAttr{
			Name: "extra",
		},
	})

	diags = node.validateResource(ctx)
	if !diags.HasErrors() {
		t.Fatal("no error for invalid depends_on")
	}
	if got, want := diags.Err().Error(), "Invalid depends_on reference"; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:  %s\nwant: Message containing %q", got, want)
	}

	// Test for handling an unknown root without attribute, like a
	// typo that omits the dot inbetween "path.module".
	rc.DependsOn = append(rc.DependsOn, hcl.Traversal{
		hcl.TraverseRoot{
			Name: "pathmodule",
		},
	})

	diags = node.validateResource(ctx)
	if !diags.HasErrors() {
		t.Fatal("no error for invalid depends_on")
	}
	if got, want := diags.Err().Error(), "Invalid depends_on reference"; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:  %s\nwant: Message containing %q", got, want)
	}
}

func TestNodeValidatableResource_ValidateResource_invalidIgnoreChangesNonexistent(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		return providers.ValidateResourceConfigResponse{}
	}

	// We'll check a _valid_ config first, to make sure we're not failing
	// for some other reason, and then make it invalid.
	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "test_object",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
		Managed: &configs.ManagedResource{
			IgnoreChanges: []hcl.Traversal{
				{
					hcl.TraverseAttr{
						Name: "test_string",
					},
				},
			},
		},
	}
	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:             mustConfigResourceAddr("test_foo.bar"),
			Config:           rc,
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	ctx.ProviderSchemaSchema = mp.GetProviderSchema()
	ctx.ProviderProvider = p

	diags := node.validateResource(ctx)
	if diags.HasErrors() {
		t.Fatalf("error for supposedly-valid config: %s", diags.ErrWithWarnings())
	}

	// Now we'll make it invalid by attempting to ignore a nonexistent
	// attribute.
	rc.Managed.IgnoreChanges = append(rc.Managed.IgnoreChanges, hcl.Traversal{
		hcl.TraverseAttr{
			Name: "nonexistent",
		},
	})

	diags = node.validateResource(ctx)
	if !diags.HasErrors() {
		t.Fatal("no error for invalid ignore_changes")
	}
	if got, want := diags.Err().Error(), "Unsupported attribute: This object has no argument, nested block, or exported attribute named \"nonexistent\""; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:  %s\nwant: Message containing %q", got, want)
	}
}

func TestNodeValidatableResource_ValidateResource_invalidIgnoreChangesComputed(t *testing.T) {
	// construct a schema with a computed attribute
	ms := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"test_string": {
				Type:     cty.String,
				Optional: true,
			},
			"computed_string": {
				Type:     cty.String,
				Computed: true,
				Optional: false,
			},
		},
	}

	mp := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Block: ms},
			ResourceTypes: map[string]providers.Schema{
				"test_object": providers.Schema{Block: ms},
			},
		},
	}

	mp.ValidateResourceConfigFn = func(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
		return providers.ValidateResourceConfigResponse{}
	}

	// We'll check a _valid_ config first, to make sure we're not failing
	// for some other reason, and then make it invalid.
	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "test_object",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
		Managed: &configs.ManagedResource{
			IgnoreChanges: []hcl.Traversal{
				{
					hcl.TraverseAttr{
						Name: "test_string",
					},
				},
			},
		},
	}
	node := NodeValidatableResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:             mustConfigResourceAddr("test_foo.bar"),
			Config:           rc,
			ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		},
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	ctx.ProviderSchemaSchema = mp.GetProviderSchema()
	ctx.ProviderProvider = p

	diags := node.validateResource(ctx)
	if diags.HasErrors() {
		t.Fatalf("error for supposedly-valid config: %s", diags.ErrWithWarnings())
	}

	// Now we'll make it invalid by attempting to ignore a computed
	// attribute.
	rc.Managed.IgnoreChanges = append(rc.Managed.IgnoreChanges, hcl.Traversal{
		hcl.TraverseAttr{
			Name: "computed_string",
		},
	})

	diags = node.validateResource(ctx)
	if diags.HasErrors() {
		t.Fatalf("got unexpected error: %s", diags.ErrWithWarnings())
	}
	if got, want := diags.ErrWithWarnings().Error(), `Redundant ignore_changes element: Adding an attribute name to ignore_changes tells Terraform to ignore future changes to the argument in configuration after the object has been created, retaining the value originally configured.

The attribute computed_string is decided by the provider alone and therefore there can be no configured value to compare with. Including this attribute in ignore_changes has no effect. Remove the attribute from ignore_changes to quiet this warning.`; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:  %s\nwant: Message containing %q", got, want)
	}
}

func Test_validateResourceForbiddenEphemeralValues(t *testing.T) {
	simpleAttrs := map[string]*configschema.Attribute{
		"input":    {Type: cty.String, Optional: true},
		"input_wo": {Type: cty.String, Optional: true, WriteOnly: true},
	}

	dynAttrs := map[string]*configschema.Attribute{
		"input":    {Type: cty.String, Optional: true},
		"input_wo": {Type: cty.String, Optional: true, WriteOnly: true},
		"dyn":      {Type: cty.DynamicPseudoType, Optional: true},
		"dyn_wo":   {Type: cty.DynamicPseudoType, Optional: true, WriteOnly: true},
	}

	allAttrs := map[string]*configschema.Attribute{
		"input":    {Type: cty.String, Optional: true},
		"input_wo": {Type: cty.String, Optional: true, WriteOnly: true},
		"dyn":      {Type: cty.DynamicPseudoType, Optional: true},
		"dyn_wo":   {Type: cty.DynamicPseudoType, Optional: true, WriteOnly: true},
		"nested_single_attr": {
			NestedType: &configschema.Object{
				Nesting:    configschema.NestingSingle,
				Attributes: dynAttrs,
			},
			Optional: true,
		},
		"nested_list_attr": {
			NestedType: &configschema.Object{
				Nesting:    configschema.NestingList,
				Attributes: dynAttrs,
			},
			Optional: true,
		},
		"nested_set_attr": {
			NestedType: &configschema.Object{
				Nesting: configschema.NestingSet,
				Attributes: map[string]*configschema.Attribute{
					"input": {Type: cty.String, Optional: true},
				},
			},
			Optional: true,
		},
		"nested_single_attr_wo": {
			NestedType: &configschema.Object{
				Nesting:    configschema.NestingSingle,
				Attributes: simpleAttrs,
			},
			Optional:  true,
			WriteOnly: true,
		},
		"nested_list_attr_wo": {
			NestedType: &configschema.Object{
				Nesting:    configschema.NestingList,
				Attributes: dynAttrs,
			},
			Optional:  true,
			WriteOnly: true,
		},
		"nested_set_attr_wo": {
			NestedType: &configschema.Object{
				Nesting: configschema.NestingSet,
				Attributes: map[string]*configschema.Attribute{
					"input": {Type: cty.String, Optional: true},
				},
			},
			Optional:  true,
			WriteOnly: true,
		},
	}

	schema := &configschema.Block{
		Attributes: allAttrs,
		BlockTypes: map[string]*configschema.NestedBlock{
			"single": {
				Block: configschema.Block{
					Attributes: dynAttrs,
				},
				Nesting: configschema.NestingSingle,
			},
			"list": {
				Block: configschema.Block{
					Attributes: dynAttrs,
				},
				Nesting: configschema.NestingList,
			},
			"set": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"input": {Type: cty.String, Optional: true},
					},
				},
				Nesting: configschema.NestingSet,
			},
			"map": {
				Block: configschema.Block{
					Attributes: simpleAttrs,
				},
				Nesting: configschema.NestingMap,
			},
		},
	}

	if err := schema.InternalValidate(); err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		obj   cty.Value
		valid bool
	}

	tests := map[string]testCase{
		"wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"input_wo": cty.StringVal("wo").Mark(marks.Ephemeral),
			}),
			valid: true,
		},
		"not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"input": cty.StringVal("wo").Mark(marks.Ephemeral),
			}),
			valid: false,
		},
		"dyn_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"dyn_wo": cty.StringVal("wo").Mark(marks.Ephemeral),
			}),
			valid: true,
		},
		"dyn_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"dyn": cty.StringVal("wo").Mark(marks.Ephemeral),
			}),
			valid: false,
		},
		"nested_dyn_wo": {
			// an ephemeral mark within a dynamic attribute is valid if the entire
			// attr is write-only
			obj: cty.ObjectVal(map[string]cty.Value{
				"dyn_wo": cty.ObjectVal(map[string]cty.Value{
					"ephem": cty.StringVal("wo").Mark(marks.Ephemeral),
				}),
			}),
			valid: true,
		},
		"nested_nested_dyn_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"dyn_wo": cty.ObjectVal(map[string]cty.Value{
					"nested": cty.ObjectVal(map[string]cty.Value{
						"ephem": cty.StringVal("wo").Mark(marks.Ephemeral),
					}),
				}),
			}),
			valid: true,
		},
		"nested_dyn_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"dyn": cty.ObjectVal(map[string]cty.Value{
					"ephem": cty.StringVal("wo").Mark(marks.Ephemeral),
				}),
			}),
			valid: false,
		},
		"nested_single_attr_attr_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"nested_single_attr": cty.ObjectVal(map[string]cty.Value{
					"input_wo": cty.StringVal("wo").Mark(marks.Ephemeral),
				}),
			}),
			valid: true,
		},
		"nested_single_attr_attr_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"nested_single_attr": cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				}),
			}),
			valid: false,
		},
		"nested_single_attr_wo_not_wo_attr": {
			// we can assign an ephemeral to input because the outer
			// nested_single_attr_wo attribute is write-only
			obj: cty.ObjectVal(map[string]cty.Value{
				"nested_single_attr_wo": cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				}),
			}),
			valid: true,
		},
		"nested_set_attr": {
			// there is no possible input_wo because the schema validated that
			// it cannot exist
			obj: cty.ObjectVal(map[string]cty.Value{
				"nested_set_attr": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				})}),
			}),
			valid: false,
		},
		"nested_set_attr_wo": {
			// assigning an ephemeral to input is valid, because the outer set is write-only
			obj: cty.ObjectVal(map[string]cty.Value{
				"nested_set_attr_wo": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				})}),
			}),
			valid: true,
		},
		"nested_list_attr_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"nested_list_attr": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				})}),
			}),
			valid: false,
		},
		"nested_list_attr_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"nested_list_attr_wo": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				})}),
			}),
			valid: true,
		},

		"single_block_attr_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"input_wo": cty.StringVal("wo").Mark(marks.Ephemeral),
				}),
			}),
			valid: true,
		},
		"single_block_attr_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				}),
			}),
			valid: false,
		},
		"single_block_dyn_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"dyn_wo": cty.ObjectVal(map[string]cty.Value{
						"ephem": cty.StringVal("wo").Mark(marks.Ephemeral),
					}),
				}),
			}),
			valid: true,
		},
		"single_block_dyn_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"single": cty.ObjectVal(map[string]cty.Value{
					"dyn": cty.ObjectVal(map[string]cty.Value{
						"ephem": cty.StringVal("wo").Mark(marks.Ephemeral),
					}),
				}),
			}),
			valid: false,
		},
		"list_block_attr_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"input_wo": cty.StringVal("wo").Mark(marks.Ephemeral),
				})}),
			}),
			valid: true,
		},
		"list_block_attr_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				})}),
			}),
			valid: false,
		},
		"list_block_dyn_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"dyn_wo": cty.ObjectVal(map[string]cty.Value{
						"ephem": cty.StringVal("wo").Mark(marks.Ephemeral),
					}),
				})}),
			}),
			valid: true,
		},
		"list_block_dyn_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"dyn": cty.ObjectVal(map[string]cty.Value{
						"ephem": cty.StringVal("wo").Mark(marks.Ephemeral),
					}),
				})}),
			}),
			valid: false,
		},
		"set_block_attr_wo": {
			// the ephemeral value within a set will always transfer the mark to
			// the outer set, but set blocks cannot be write-only
			obj: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"input": cty.StringVal("wo").Mark(marks.Ephemeral),
				})}),
			}),
			valid: false,
		},
		"map_block_attr_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"test": cty.ObjectVal(map[string]cty.Value{
						"input_wo": cty.StringVal("wo").Mark(marks.Ephemeral),
					}),
				}),
			}),
			valid: true,
		},
		"map_block_attr_not_wo": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"test": cty.ObjectVal(map[string]cty.Value{
						"input": cty.StringVal("wo").Mark(marks.Ephemeral),
					}),
				}),
			}),
			valid: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			val, err := schema.CoerceValue(tc.obj)
			if err != nil {
				t.Fatal(err)
			}
			diags := validateResourceForbiddenEphemeralValues(nil, val, schema)
			switch {
			case tc.valid && diags.HasErrors():
				t.Fatal("unexpected diags:", diags.ErrWithWarnings())
			case !tc.valid && !diags.HasErrors():
				t.Fatal("expected diagnostics, got none")
			}
		})
	}
}
