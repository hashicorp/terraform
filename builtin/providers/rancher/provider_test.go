package rancher

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
		"powerdns": testAccProvider,
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
	if v := os.Getenv("RANCHER_URL"); v == "" {
		t.Fatal("RANCHER_URL must be set for acceptance tests")
	}

	if v := os.Getenv("RANCHER_ACCESS_KEY"); v == "" {
		t.Fatal("RANCHER_ACCESS_KEY must be set for acceptance tests")
	}

	if v := os.Getenv("RANCHER_SECRET_KEY"); v == "" {
		t.Fatal("RANCHER_SECRET_KEY must be set for acceptance tests")
	}
}
