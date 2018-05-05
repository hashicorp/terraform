package terraform

import (
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
)

func TestEvalValidateResource_managedResource(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateResourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		expected := "aws_instance"
		if rt != expected {
			t.Fatalf("expected: %s, got: %s", expected, rt)
		}
		expected = "bar"
		val, _ := c.Get("foo")
		if val != expected {
			t.Fatalf("expected: %s, got: %s", expected, val)
		}
		return
	}

	p := ResourceProvider(mp)
	rc := &configs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "aws_instance",
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"foo": cty.StringVal("bar"),
		}),
	}
	node := &EvalValidateResource{
		Addr: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			},
		},
		Provider: &p,
		Config:   rc,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mp.ValidateResourceCalled {
		t.Fatal("Expected ValidateResource to be called, but it was not!")
	}
}

func TestEvalValidateResource_dataSource(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateDataSourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		expected := "aws_ami"
		if rt != expected {
			t.Fatalf("expected: %s, got: %s", expected, rt)
		}
		expected = "bar"
		val, _ := c.Get("foo")
		if val != expected {
			t.Fatalf("expected: %s, got: %s", expected, val)
		}
		return
	}

	p := ResourceProvider(mp)
	rc := &configs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "aws_ami",
		Name: "foo",
		Config: configs.SynthBody("", map[string]cty.Value{
			"foo": cty.StringVal("bar"),
		}),
	}

	node := &EvalValidateResource{
		Addr: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "aws_ami",
				Name: "foo",
			},
		},
		Provider: &p,
		Config:   rc,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mp.ValidateDataSourceCalled {
		t.Fatal("Expected ValidateDataSource to be called, but it was not!")
	}
}

func TestEvalValidateResource_validReturnsNilError(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateResourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		return
	}

	p := ResourceProvider(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "aws_instance",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
	}
	node := &EvalValidateResource{
		Addr: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			},
		},
		Provider: &p,
		Config:   rc,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err != nil {
		t.Fatalf("Expected nil error, got: %s", err)
	}
}

func TestEvalValidateResource_warningsAndErrorsPassedThrough(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateResourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		ws = append(ws, "warn")
		es = append(es, errors.New("err"))
		return
	}

	p := ResourceProvider(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "aws_instance",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
	}
	node := &EvalValidateResource{
		Addr: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			},
		},
		Provider: &p,
		Config:   rc,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err == nil {
		t.Fatal("Expected an error, got none!")
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(err)
	bySeverity := map[tfdiags.Severity]tfdiags.Diagnostics{}
	for _, diag := range diags {
		bySeverity[diag.Severity()] = append(bySeverity[diag.Severity()], diag)
	}
	if len(bySeverity[tfdiags.Warning]) != 1 || bySeverity[tfdiags.Warning][0].Description().Summary != "warn" {
		t.Fatalf("Expected 1 warning 'warn', got: %#v", bySeverity[tfdiags.Warning])
	}
	if len(bySeverity[tfdiags.Error]) != 1 || bySeverity[tfdiags.Error][0].Description().Summary != "err" {
		t.Fatalf("Expected 1 error 'err', got: %#v", bySeverity[tfdiags.Error])
	}
}

func TestEvalValidateResource_ignoreWarnings(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateResourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		ws = append(ws, "warn")
		return
	}

	p := ResourceProvider(mp)
	rc := &configs.Resource{
		Mode:   addrs.ManagedResourceMode,
		Type:   "aws_instance",
		Name:   "foo",
		Config: configs.SynthBody("", map[string]cty.Value{}),
	}
	node := &EvalValidateResource{
		Addr: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			},
		},
		Provider: &p,
		Config:   rc,

		IgnoreWarnings: true,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err != nil {
		t.Fatalf("Expected no error, got: %s", err)
	}
}

func TestEvalValidateProvisioner_valid(t *testing.T) {
	mp := &MockResourceProvisioner{}
	var p ResourceProvisioner = mp
	ctx := &MockEvalContext{}

	schema := &configschema.Block{}

	node := &EvalValidateProvisioner{
		ResourceAddr: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			},
		},
		Provisioner: &p,
		Schema:      &schema,
		Config: &configs.Provisioner{
			Type:   "baz",
			Config: hcl.EmptyBody(),
		},
		ConnConfig: &configs.Connection{
			Type:   "ssh",
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

	if !mp.ValidateCalled {
		t.Fatalf("p.Validate not called")
	}
}

func TestEvalValidateProvisioner_warning(t *testing.T) {
	mp := &MockResourceProvisioner{}
	var p ResourceProvisioner = mp
	ctx := &MockEvalContext{}

	schema := &configschema.Block{}

	node := &EvalValidateProvisioner{
		ResourceAddr: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			},
		},
		Provisioner: &p,
		Schema:      &schema,
		Config: &configs.Provisioner{
			Type:   "baz",
			Config: hcl.EmptyBody(),
		},
		ConnConfig: &configs.Connection{
			Type:   "ssh",
			Config: hcl.EmptyBody(),
		},
	}

	mp.ValidateReturnWarns = []string{"foo is deprecated"}

	_, err := node.Eval(ctx)
	if err == nil {
		t.Fatalf("node.Eval succeeded; want error")
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(err)
	if len(diags) != 1 {
		t.Fatalf("wrong number of diagsnostics in %#v; want one warning", diags)
	}

	if got, want := diags[0].Description().Summary, mp.ValidateReturnWarns[0]; got != want {
		t.Fatalf("wrong warning %q; want %q", got, want)
	}
}

func TestEvalValidateProvisioner_connectionInvalid(t *testing.T) {
	var p ResourceProvisioner = &MockResourceProvisioner{}
	ctx := &MockEvalContext{}

	schema := &configschema.Block{}

	node := &EvalValidateProvisioner{
		ResourceAddr: addrs.ResourceInstance{
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			},
		},
		Provisioner: &p,
		Schema:      &schema,
		Config: &configs.Provisioner{
			Type:   "baz",
			Config: hcl.EmptyBody(),
		},
		ConnConfig: &configs.Connection{
			Type: "ssh",
			Config: configs.SynthBody("", map[string]cty.Value{
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
		t.Fatalf("wrong number of diagsnostics in %#v; want two errors", diags)
	}

	errStr := diags.Err().Error()
	if !(strings.Contains(errStr, "bananananananana") && strings.Contains(errStr, "bazaz")) {
		t.Fatalf("wrong errors %q; want something about each of our invalid connInfo keys", errStr)
	}
}
