package dyn

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
		"dyn": testAccProvider,
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
	if v := os.Getenv("DYN_CUSTOMER_NAME"); v == "" {
		t.Fatal("DYN_CUSTOMER_NAME must be set for acceptance tests")
	}

	if v := os.Getenv("DYN_USERNAME"); v == "" {
		t.Fatal("DYN_USERNAME must be set for acceptance tests")
	}

	if v := os.Getenv("DYN_PASSWORD"); v == "" {
		t.Fatal("DYN_PASSWORD must be set for acceptance tests.")
	}

	if v := os.Getenv("DYN_ZONE"); v == "" {
		t.Fatal("DYN_ZONE must be set for acceptance tests. The domain is used to ` and destroy record against.")
	}
}
