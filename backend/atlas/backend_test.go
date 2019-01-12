package atlas

import (
	"os"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/backend"
)

func TestImpl(t *testing.T) {
	var _ backend.Backend = new(Backend)
	var _ backend.CLI = new(Backend)
}

func TestConfigure_envAddr(t *testing.T) {
	defer os.Setenv("ATLAS_ADDRESS", os.Getenv("ATLAS_ADDRESS"))
	os.Setenv("ATLAS_ADDRESS", "http://foo.com")

	b := New()
	diags := b.Configure(cty.ObjectVal(map[string]cty.Value{
		"name":         cty.StringVal("foo/bar"),
		"address":      cty.NullVal(cty.String),
		"access_token": cty.StringVal("placeholder"),
	}))
	for _, diag := range diags {
		t.Error(diag)
	}

	if got, want := b.stateClient.Server, "http://foo.com"; got != want {
		t.Fatalf("wrong URL %#v; want %#v", got, want)
	}
}

func TestConfigure_envToken(t *testing.T) {
	defer os.Setenv("ATLAS_TOKEN", os.Getenv("ATLAS_TOKEN"))
	os.Setenv("ATLAS_TOKEN", "foo")

	b := New()
	diags := b.Configure(cty.ObjectVal(map[string]cty.Value{
		"name":         cty.StringVal("foo/bar"),
		"address":      cty.NullVal(cty.String),
		"access_token": cty.NullVal(cty.String),
	}))
	for _, diag := range diags {
		t.Error(diag)
	}

	if got, want := b.stateClient.AccessToken, "foo"; got != want {
		t.Fatalf("wrong access token %#v; want %#v", got, want)
	}
}
