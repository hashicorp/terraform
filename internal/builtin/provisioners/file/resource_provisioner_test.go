package file

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/zclconf/go-cty/cty"
)

func TestResourceProvider_Validate_good_source(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"source":      cty.StringVal("/tmp/foo"),
		"destination": cty.StringVal("/tmp/bar"),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: v,
	})

	if len(resp.Diagnostics) > 0 {
		t.Fatal(resp.Diagnostics.ErrWithWarnings())
	}
}

func TestResourceProvider_Validate_good_content(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"content":     cty.StringVal("value to copy"),
		"destination": cty.StringVal("/tmp/bar"),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: v,
	})

	if len(resp.Diagnostics) > 0 {
		t.Fatal(resp.Diagnostics.ErrWithWarnings())
	}
}

func TestResourceProvider_Validate_good_unknown_variable_value(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"content":     cty.UnknownVal(cty.String),
		"destination": cty.StringVal("/tmp/bar"),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: v,
	})

	if len(resp.Diagnostics) > 0 {
		t.Fatal(resp.Diagnostics.ErrWithWarnings())
	}
}

func TestResourceProvider_Validate_bad_not_destination(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"source": cty.StringVal("nope"),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: v,
	})

	if !resp.Diagnostics.HasErrors() {
		t.Fatal("Should have errors")
	}
}

func TestResourceProvider_Validate_bad_no_source(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"destination": cty.StringVal("/tmp/bar"),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: v,
	})

	if !resp.Diagnostics.HasErrors() {
		t.Fatal("Should have errors")
	}
}

func TestResourceProvider_Validate_bad_to_many_src(t *testing.T) {
	v := cty.ObjectVal(map[string]cty.Value{
		"source":      cty.StringVal("nope"),
		"content":     cty.StringVal("vlue to copy"),
		"destination": cty.StringVal("/tmp/bar"),
	})

	resp := New().ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{
		Config: v,
	})

	if !resp.Diagnostics.HasErrors() {
		t.Fatal("Should have errors")
	}
}

// Validate that Stop can Close can be called even when not provisioning.
func TestResourceProvisioner_StopClose(t *testing.T) {
	p := New()
	p.Stop()
	p.Close()
}

func TestResourceProvisioner_connectionRequired(t *testing.T) {
	p := New()
	resp := p.ProvisionResource(provisioners.ProvisionResourceRequest{})
	if !resp.Diagnostics.HasErrors() {
		t.Fatal("expected error")
	}

	got := resp.Diagnostics.Err().Error()
	if !strings.Contains(got, "Missing connection") {
		t.Fatalf("expected 'Missing connection' error: got %q", got)
	}
}
