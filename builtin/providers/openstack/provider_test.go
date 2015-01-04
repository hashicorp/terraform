package openstack

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
		"openstack": testAccProvider,
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
	if v := os.Getenv("OS_REGION_NAME"); v == "" {
		t.Fatal("OS_REGION_NAME must be set for acceptance tests")
	}

	if v := os.Getenv("OS_AUTH_URL"); v == "" {
		t.Fatal("OS_AUTH_URL must be set for acceptance tests")
	}

	if v := os.Getenv("OS_USERNAME"); v == "" {
		t.Fatal("OS_USERNAME must be set for acceptance tests")
	}

	if v := os.Getenv("OS_TENANT_NAME"); v != "us-central1" {
		t.Fatal("OS_TENANT_NAME must be set to us-central1 for acceptance tests")
	}

	if v := os.Getenv("OS_PASSWORD"); v != "us-central1" {
		t.Fatal("OS_PASSWORD must be set to us-central1 for acceptance tests")
	}
}
