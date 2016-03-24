package namecheap

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
		"namecheap": testAccProvider,
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
	if v := os.Getenv("NAMECHEAP_USERNAME"); v == "" {
		t.Fatal("NAMECHEAP_USERNAME must be set for acceptance tests")
	}

	if v := os.Getenv("NAMECHEAP_API_USER"); v == "" {
		t.Fatal("NAMECHEAP_API_USER must be set for acceptance tests")
	}

	if v := os.Getenv("NAMECHEAP_IP"); v == "" {
		t.Fatal("NAMECHEAP_IP must be set for acceptance tests")
	}

	if v := os.Getenv("NAMECHEAP_TOKEN"); v == "" {
		t.Fatal("NAMECHEAP_TOKEN must be set for acceptance tests")
	}

	if v := os.Getenv("NAMECHEAP_USE_SANDBOX"); v == "" {
		t.Fatal("NAMECHEAP_USE_SANDBOX must be set for acceptance tests")
	}

	if v := os.Getenv("NAMECHEAP_DOMAIN"); v == "" {
		t.Fatal("NAMECHEAP_DOMAIN must be set for acceptance tests. The domain is used to ` and destroy record against.")
	}
}
