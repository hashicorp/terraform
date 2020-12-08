package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
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

	diags := node.validateProvisioner(ctx, pc, false, false)
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

	diags := node.validateProvisioner(ctx, pc, false, false)
	if len(diags) != 1 {
		t.Fatalf("wrong number of diagnostics in %s; want one warning", diags.ErrWithWarnings())
	}

	if got, want := diags[0].Description().Summary, mp.ValidateProvisionerConfigResponse.Diagnostics[0].Description().Summary; got != want {
		t.Fatalf("wrong warning %q; want %q", got, want)
	}
}

func TestNodeValidatableResource_ValidateProvisioner__conntectionInvalid(t *testing.T) {
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

	diags := node.validateProvisioner(ctx, pc, false, false)
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
