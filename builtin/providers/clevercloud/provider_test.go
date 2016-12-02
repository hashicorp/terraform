package clevercloud

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"clevercloud": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("CLEVERCLOUD_ENDPOINT"); v == "" {
		t.Fatal("CLEVERCLOUD_ENDPOINT must be set for acceptance tests")
	}
	if v := os.Getenv("CLEVERCLOUD_AUTH_TOKEN"); v == "" {
		t.Fatal("CLEVERCLOUD_AUTH_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("CLEVERCLOUD_AUTH_SECRET"); v == "" {
		t.Fatal("CLEVERCLOUD_AUTH_SECRET must be set for acceptance tests")
	}
}
