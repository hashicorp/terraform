package dme

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
		// provider is called terraform-provider-dme ie dme
		"dme": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderImpl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("DME_SKEY"); v == "" {
		t.Fatal("DME_SKEY must be set for acceptance tests")
	}

	if v := os.Getenv("DME_AKEY"); v == "" {
		t.Fatal("DME_AKEY must be set for acceptance tests")
	}

	if v := os.Getenv("DME_DOMAINID"); v == "" {
		t.Fatal("DME_DOMAINID must be set for acceptance tests")
	}

	if v := os.Getenv("DME_USESANDBOX"); v == "" {
		t.Fatal("DME_USESANDBOX must be set for acceptance tests. Use the strings 'true' or 'false'.")
	}
}
