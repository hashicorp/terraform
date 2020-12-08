package terraform

import (
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestEvalValidateResource_managedResource(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		if got, want := req.TypeName, "test_object"; got != want {
			t.Fatalf("wrong resource type\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_string"), cty.StringVal("bar"); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_string\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_number"), cty.NumberIntVal(2); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_number\ngot:  %#v\nwant: %#v", got, want)
		}
		return providers.ValidateResourceTypeConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_object",
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("bar"),
			"test_number": cty.NumberIntVal(2).Mark("sensitive"),
		}),
	}
	node := &EvalValidateResource{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "foo",
		},
		Provider:       &p,
		Config:         rc,
		ProviderSchema: &mp.GetSchemaReturn,
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	err := node.Validate(ctx)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mp.ValidateResourceTypeConfigCalled {
		t.Fatal("Expected ValidateResourceTypeConfig to be called, but it was not!")
	}
}

func TestEvalValidateResource_managedResourceCount(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		if got, want := req.TypeName, "test_object"; got != want {
			t.Fatalf("wrong resource type\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_string"), cty.StringVal("bar"); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_string\ngot:  %#v\nwant: %#v", got, want)
		}
		return providers.ValidateResourceTypeConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode:  addrs.ManagedResourceMode,
		Type:  "test_object",
		Name:  "foo",
		Count: hcltest.MockExprLiteral(cty.NumberIntVal(2)),
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("bar"),
		}),
	}
	node := &EvalValidateResource{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "foo",
		},
		Provider:       &p,
		Config:         rc,
		ProviderSchema: &mp.GetSchemaReturn,
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	err := node.Validate(ctx)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mp.ValidateResourceTypeConfigCalled {
		t.Fatal("Expected ValidateResourceTypeConfig to be called, but it was not!")
	}
}

func TestEvalValidateResource_dataSource(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateDataSourceConfigFn = func(req providers.ValidateDataSourceConfigRequest) providers.ValidateDataSourceConfigResponse {
		if got, want := req.TypeName, "test_object"; got != want {
			t.Fatalf("wrong resource type\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_string"), cty.StringVal("bar"); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_string\ngot:  %#v\nwant: %#v", got, want)
		}
		if got, want := req.Config.GetAttr("test_number"), cty.NumberIntVal(2); !got.RawEquals(want) {
			t.Fatalf("wrong value for test_number\ngot:  %#v\nwant: %#v", got, want)
		}
		return providers.ValidateDataSourceConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "test_object",
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("bar"),
			"test_number": cty.NumberIntVal(2).Mark("sensitive"),
		}),
	}

	node := &EvalValidateResource{
		Addr: addrs.Resource{
			Mode: addrs.DataResourceMode,
			Type: "aws_ami",
			Name: "foo",
		},
		Provider:       &p,
		Config:         rc,
		ProviderSchema: &mp.GetSchemaReturn,
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	err := node.Validate(ctx)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mp.ValidateDataSourceConfigCalled {
		t.Fatal("Expected ValidateDataSourceConfig to be called, but it was not!")
	}
}

func TestEvalValidateResource_validReturnsNilError(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		return providers.ValidateResourceTypeConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "test_object",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
	}
	node := &EvalValidateResource{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_object",
			Name: "foo",
		},
		Provider:       &p,
		Config:         rc,
		ProviderSchema: &mp.GetSchemaReturn,
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	err := node.Validate(ctx)
	if err != nil {
		t.Fatalf("Expected nil error, got: %s", err)
	}
}

func TestEvalValidateResource_warningsAndErrorsPassedThrough(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.SimpleWarning("warn"))
		diags = diags.Append(errors.New("err"))
		return providers.ValidateResourceTypeConfigResponse{
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
	node := &EvalValidateResource{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_object",
			Name: "foo",
		},
		Provider:       &p,
		Config:         rc,
		ProviderSchema: &mp.GetSchemaReturn,
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	err := node.Validate(ctx)
	if err == nil {
		t.Fatal("unexpected success; want error")
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(err)
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

func TestEvalValidateResource_invalidDependsOn(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		return providers.ValidateResourceTypeConfigResponse{}
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
	node := &EvalValidateResource{
		Addr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "foo",
		},
		Provider:       &p,
		Config:         rc,
		ProviderSchema: &mp.GetSchemaReturn,
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	diags := node.Validate(ctx)
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

	diags = node.Validate(ctx)
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

	diags = node.Validate(ctx)
	if !diags.HasErrors() {
		t.Fatal("no error for invalid depends_on")
	}
	if got, want := diags.Err().Error(), "Invalid depends_on reference"; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:  %s\nwant: Message containing %q", got, want)
	}
}
