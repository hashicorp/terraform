package vcd

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
		"vcd": testAccProvider,
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
	if v := os.Getenv("VCD_USER"); v == "" {
		t.Fatal("VCD_USER must be set for acceptance tests")
	}
	if v := os.Getenv("VCD_PASSWORD"); v == "" {
		t.Fatal("VCD_PASSWORD must be set for acceptance tests")
	}
	if v := os.Getenv("VCD_ORG"); v == "" {
		t.Fatal("VCD_ORG must be set for acceptance tests")
	}
	if v := os.Getenv("VCD_URL"); v == "" {
		t.Fatal("VCD_URL must be set for acceptance tests")
	}
	if v := os.Getenv("VCD_EDGE_GATEWAY"); v == "" {
		t.Fatal("VCD_EDGE_GATEWAY must be set for acceptance tests")
	}
	if v := os.Getenv("VCD_VDC"); v == "" {
		t.Fatal("VCD_VDC must be set for acceptance tests")
	}
}
