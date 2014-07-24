package dnsimple

import (
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *ResourceProvider

func init() {
	testAccProvider = new(ResourceProvider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"dnsimple": testAccProvider,
	}
}

func TestResourceProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = new(ResourceProvider)
}

func TestResourceProvider_Configure(t *testing.T) {
	rp := new(ResourceProvider)
	var expectedToken string
	var expectedEmail string

	if v := os.Getenv("DNSIMPLE_EMAIL"); v != "" {
		expectedEmail = v
	} else {
		expectedEmail = "foo"
	}

	if v := os.Getenv("DNSIMPLE_TOKEN"); v != "" {
		expectedToken = v
	} else {
		expectedToken = "foo"
	}

	raw := map[string]interface{}{
		"token": expectedToken,
		"email": expectedEmail,
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := Config{
		Token: expectedToken,
		Email: expectedEmail,
	}

	if !reflect.DeepEqual(rp.Config, expected) {
		t.Fatalf("bad: %#v", rp.Config)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("DNSIMPLE_EMAIL"); v == "" {
		t.Fatal("DNSIMPLE_EMAIL must be set for acceptance tests")
	}

	if v := os.Getenv("DNSIMPLE_TOKEN"); v == "" {
		t.Fatal("DNSIMPLE_TOKEN must be set for acceptance tests")
	}

	if v := os.Getenv("DNSIMPLE_DOMAIN"); v == "" {
		t.Fatal("DNSIMPLE_DOMAIN must be set for acceptance tests. The domain is used to ` and destroy record against.")
	}
}
