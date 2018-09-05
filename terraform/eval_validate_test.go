package terraform

import (
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
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
		return providers.ValidateResourceTypeConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_object",
		Name: "foo",
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

	_, err := node.Eval(ctx)
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

	_, err := node.Eval(ctx)
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
		return providers.ValidateDataSourceConfigResponse{}
	}

	p := providers.Interface(mp)
	rc := &configs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "test_object",
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("bar"),
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

	_, err := node.Eval(ctx)
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

	_, err := node.Eval(ctx)
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

	_, err := node.Eval(ctx)
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

func TestEvalValidateResource_ignoreWarnings(t *testing.T) {
	mp := simpleMockProvider()
	mp.ValidateResourceTypeConfigFn = func(req providers.ValidateResourceTypeConfigRequest) providers.ValidateResourceTypeConfigResponse {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.SimpleWarning("warn"))
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
			Type: "test-object",
			Name: "foo",
		},
		Provider:       &p,
		Config:         rc,
		ProviderSchema: &mp.GetSchemaReturn,

		IgnoreWarnings: true,
	}

	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	_, err := node.Eval(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %s", err)
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

	_, err := node.Eval(ctx)
	if err != nil {
		t.Fatalf("error for supposedly-valid config: %s", err)
	}

	// No we'll make it invalid, but adding additional traversal steps
	// on the end of what we're referencing. This is intended to catch the
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

	_, err = node.Eval(ctx)
	if err == nil {
		t.Fatal("no error for invalid depends_on")
	}
	if got, want := err.Error(), "Invalid depends_on reference"; !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:  %s\nwant: Message containing %q", got, want)
	}
}

func TestEvalValidateProvisioner_valid(t *testing.T) {
	mp := &MockProvisioner{}
	var p provisioners.Interface = mp
	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	schema := &configschema.Block{}

	node := &EvalValidateProvisioner{
		ResourceAddr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "bar",
		},
		Provisioner: &p,
		Schema:      &schema,
		Config: &configs.Provisioner{
			Type:   "baz",
			Config: hcl.EmptyBody(),
		},
		ConnConfig: &configs.Connection{
			//Type:   "ssh",
			Config: hcl.EmptyBody(),
		},
	}

	result, err := node.Eval(ctx)
	if err != nil {
		t.Fatalf("node.Eval failed: %s", err)
	}
	if result != nil {
		t.Errorf("node.Eval returned non-nil result")
	}

	if !mp.ValidateProvisionerConfigCalled {
		t.Fatalf("p.ValidateProvisionerConfig not called")
	}
}

func TestEvalValidateProvisioner_warning(t *testing.T) {
	mp := &MockProvisioner{}
	var p provisioners.Interface = mp
	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"type": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	node := &EvalValidateProvisioner{
		ResourceAddr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "bar",
		},
		Provisioner: &p,
		Schema:      &schema,
		Config: &configs.Provisioner{
			Type:   "baz",
			Config: hcl.EmptyBody(),
		},
		ConnConfig: &configs.Connection{
			Config: configs.SynthBody("", map[string]cty.Value{
				"type": cty.StringVal("ssh"),
			}),
		},
	}

	{
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.SimpleWarning("foo is deprecated"))
		mp.ValidateProvisionerConfigResponse = provisioners.ValidateProvisionerConfigResponse{
			Diagnostics: diags,
		}
	}

	_, err := node.Eval(ctx)
	if err == nil {
		t.Fatalf("node.Eval succeeded; want error")
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(err)
	if len(diags) != 1 {
		t.Fatalf("wrong number of diagsnostics in %#v; want one warning", diags)
	}

	if got, want := diags[0].Description().Summary, mp.ValidateProvisionerConfigResponse.Diagnostics[0].Description().Summary; got != want {
		t.Fatalf("wrong warning %q; want %q", got, want)
	}
}

func TestEvalValidateProvisioner_connectionInvalid(t *testing.T) {
	var p provisioners.Interface = &MockProvisioner{}
	ctx := &MockEvalContext{}
	ctx.installSimpleEval()

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"type": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	node := &EvalValidateProvisioner{
		ResourceAddr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "bar",
		},
		Provisioner: &p,
		Schema:      &schema,
		Config: &configs.Provisioner{
			Type:   "baz",
			Config: hcl.EmptyBody(),
		},
		ConnConfig: &configs.Connection{
			Config: configs.SynthBody("", map[string]cty.Value{
				"type":             cty.StringVal("ssh"),
				"bananananananana": cty.StringVal("foo"),
				"bazaz":            cty.StringVal("bar"),
			}),
		},
	}

	_, err := node.Eval(ctx)
	if err == nil {
		t.Fatalf("node.Eval succeeded; want error")
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(err)
	if len(diags) != 2 {
		t.Fatalf("wrong number of diagnostics; want two errors\n\n%s", diags.Err())
	}

	errStr := diags.Err().Error()
	if !(strings.Contains(errStr, "bananananananana") && strings.Contains(errStr, "bazaz")) {
		t.Fatalf("wrong errors %q; want something about each of our invalid connInfo keys", errStr)
	}
}
