package openstack

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var (
	OS_REGION_NAME = ""
	OS_POOL_NAME   = ""
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
	v := os.Getenv("OS_AUTH_URL")
	if v == "" {
		t.Fatal("OS_AUTH_URL must be set for acceptance tests")
	}

	v = os.Getenv("OS_REGION_NAME")
	if v != "" {
		OS_REGION_NAME = v
	}

	v1 := os.Getenv("OS_IMAGE_ID")
	v2 := os.Getenv("OS_IMAGE_NAME")

	if v1 == "" || v2 == "" {
		t.Fatal("OS_IMAGE_ID and OS_IMAGE_NAME must be set for acceptance tests")
	}

	v = os.Getenv("OS_POOL_NAME")
	if v == "" {
		t.Fatal("OS_POOL_NAME must be set for acceptance tests")
	}
	OS_POOL_NAME = v

	v1 = os.Getenv("OS_FLAVOR_ID")
	v2 = os.Getenv("OS_FLAVOR_NAME")
	if v1 == "" && v2 == "" {
		t.Fatal("OS_FLAVOR_ID or OS_FLAVOR_NAME must be set for acceptance tests")
	}

	v = os.Getenv("OS_NETWORK_ID")
	if v == "" {
		t.Fatal("OS_NETWORK_ID must be set for acceptance tests")
	}
}
