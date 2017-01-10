package openstack

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var (
	OS_EXTGW_ID    = os.Getenv("OS_EXTGW_ID")
	OS_FLAVOR_ID   = os.Getenv("OS_FLAVOR_ID")
	OS_FLAVOR_NAME = os.Getenv("OS_FLAVOR_NAME")
	OS_IMAGE_ID    = os.Getenv("OS_IMAGE_ID")
	OS_IMAGE_NAME  = os.Getenv("OS_IMAGE_NAME")
	OS_NETWORK_ID  = os.Getenv("OS_NETWORK_ID")
	OS_POOL_NAME   = os.Getenv("OS_POOL_NAME")
	OS_REGION_NAME = os.Getenv("OS_REGION_NAME")
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

	if OS_IMAGE_ID == "" || OS_IMAGE_NAME == "" {
		t.Fatal("OS_IMAGE_ID and OS_IMAGE_NAME must be set for acceptance tests")
	}

	if OS_POOL_NAME == "" {
		t.Fatal("OS_POOL_NAME must be set for acceptance tests")
	}

	if OS_FLAVOR_ID == "" && OS_FLAVOR_NAME == "" {
		t.Fatal("OS_FLAVOR_ID or OS_FLAVOR_NAME must be set for acceptance tests")
	}

	if OS_NETWORK_ID == "" {
		t.Fatal("OS_NETWORK_ID must be set for acceptance tests")
	}

	if OS_EXTGW_ID == "" {
		t.Fatal("OS_EXTGW_ID must be set for acceptance tests")
	}
}
