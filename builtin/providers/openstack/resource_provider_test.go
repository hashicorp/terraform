package openstack

import (
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *ResourceProvider

func init() {
	testAccProvider = new(ResourceProvider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"openstack": testAccProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("OS_AUTH_URL"); v == "" {
		t.Fatal("OS_AUTH_URL must be set for acceptance tests")
	}

	if v := os.Getenv("OS_USERNAME"); v == "" {
		t.Fatal("OS_USERNAME must be set for acceptance tests")
	}

	if v := os.Getenv("OS_PASSWORD"); v == "" {
		t.Fatal("OS_PASSWORD must be set for acceptance tests.")
	}

	if v := os.Getenv("OS_TENANT_NAME"); v == "" {
		t.Fatal("OS_TENANT_NAME must be set for acceptance tests.")
	}
}
