package cobbler

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccCobblerProviders map[string]terraform.ResourceProvider
var testAccCobblerProvider *schema.Provider

func init() {
	testAccCobblerProvider = Provider().(*schema.Provider)
	testAccCobblerProviders = map[string]terraform.ResourceProvider{
		"cobbler": testAccCobblerProvider,
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

func testAccCobblerPreCheck(t *testing.T) {
	v := os.Getenv("COBBLER_USERNAME")
	if v == "" {
		t.Fatal("COBBLER_USERNAME must be set for acceptance tests.")
	}

	v = os.Getenv("COBBLER_PASSWORD")
	if v == "" {
		t.Fatal("COBBLER_PASSWORD must be set for acceptance tests.")
	}

	v = os.Getenv("COBBLER_URL")
	if v == "" {
		t.Fatal("COBBLER_URL must be set for acceptance tests.")
	}
}
