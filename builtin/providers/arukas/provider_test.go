package arukas

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
		"arukas": testAccProvider,
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
	if v := os.Getenv("ARUKAS_JSON_API_TOKEN"); v == "" {
		t.Fatal("ARUKAS_JSON_API_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("ARUKAS_JSON_API_SECRET"); v == "" {
		t.Fatal("ARUKAS_JSON_API_SECRET must be set for acceptance tests")
	}
}
