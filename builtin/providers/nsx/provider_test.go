package nsx

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
		"nsx": testAccProvider,
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
	if v := os.Getenv("NSX_USER"); v == "" {
		t.Fatal("NSX_USER must be set for acceptance tests")
	}

	if v := os.Getenv("NSX_PASSWORD"); v == "" {
		t.Fatal("NSX_PASSWORD must be set for acceptance tests")
	}

	if v := os.Getenv("NSX_SERVER"); v == "" {
		t.Fatal("NSX_SERVER must be set for acceptance tests")
	}
}
