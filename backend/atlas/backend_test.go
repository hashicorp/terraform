package atlas

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestImpl(t *testing.T) {
	var _ backend.Backend = new(Backend)
	var _ backend.CLI = new(Backend)
}

func TestConfigure_envAddr(t *testing.T) {
	defer os.Setenv("ATLAS_ADDRESS", os.Getenv("ATLAS_ADDRESS"))
	os.Setenv("ATLAS_ADDRESS", "http://foo.com")

	b := New()
	err := b.Configure(terraform.NewResourceConfig(config.TestRawConfig(t, map[string]interface{}{
		"name": "foo/bar",
	})))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if b.stateClient.Server != "http://foo.com" {
		t.Fatalf("bad: %#v", b.stateClient)
	}
}

func TestConfigure_envToken(t *testing.T) {
	defer os.Setenv("ATLAS_TOKEN", os.Getenv("ATLAS_TOKEN"))
	os.Setenv("ATLAS_TOKEN", "foo")

	b := New()
	err := b.Configure(terraform.NewResourceConfig(config.TestRawConfig(t, map[string]interface{}{
		"name": "foo/bar",
	})))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if b.stateClient.AccessToken != "foo" {
		t.Fatalf("bad: %#v", b.stateClient)
	}
}
