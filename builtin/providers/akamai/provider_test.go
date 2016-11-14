package akamai

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
		"akamai": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("AKAMAI_EDGEGRID_HOST"); v == "" {
		t.Fatal("AKAMAI_EDGEGRID_HOST must be set for acceptance tests")
	}
	if v := os.Getenv("AKAMAI_EDGEGRID_ACCESS_TOKEN"); v == "" {
		t.Fatal("AKAMAI_EDGEGRID_ACCESS_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("AKAMAI_EDGEGRID_CLIENT_TOKEN"); v == "" {
		t.Fatal("AKAMAI_EDGEGRID_CLIENT_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("AKAMAI_EDGEGRID_CLIENT_SECRET"); v == "" {
		t.Fatal("AKAMAI_EDGEGRID_CLIENT_SECRET must be set for acceptance tests")
	}
}
