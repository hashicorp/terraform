package google

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
		"google": testAccProvider,
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
	if v := os.Getenv("GOOGLE_ACCOUNT_FILE"); v == "" {
		t.Fatal("GOOGLE_ACCOUNT_FILE must be set for acceptance tests")
	}

	if v := os.Getenv("GOOGLE_CLIENT_FILE"); v == "" {
		t.Fatal("GOOGLE_CLIENT_FILE must be set for acceptance tests")
	}

	if v := os.Getenv("GOOGLE_PROJECT"); v == "" {
		t.Fatal("GOOGLE_PROJECT must be set for acceptance tests")
	}
}
