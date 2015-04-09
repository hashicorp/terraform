package rackspace

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var (
	RS_REGION_NAME = ""
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"rackspace": testAccProvider,
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
	v := os.Getenv("RS_AUTH_URL")
	if v == "" {
		t.Fatal("RS_AUTH_URL must be set for acceptance tests")
	}

	v = os.Getenv("RS_REGION_NAME")
	if v == "" {
		t.Fatal("RS_REGION_NAME must be set for acceptance tests")
	}
	RS_REGION_NAME = v

	v1 := os.Getenv("RS_IMAGE_ID")
	v2 := os.Getenv("RS_IMAGE_NAME")

	if v1 == "" && v2 == "" {
		t.Fatal("RS_IMAGE_ID or RS_IMAGE_NAME must be set for acceptance tests")
	}

	v1 = os.Getenv("RS_FLAVOR_ID")
	v2 = os.Getenv("RS_FLAVOR_NAME")
	if v1 == "" && v2 == "" {
		t.Fatal("RS_FLAVOR_ID or RS_FLAVOR_NAME must be set for acceptance tests")
	}

	v = os.Getenv("RS_NETWORK_ID")
	if v == "" {
		t.Fatal("RS_NETWORK_ID must be set for acceptance tests")
	}
}
