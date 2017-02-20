package shield

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

const testRepo string = "test-repo"

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"shield": testAccProvider,
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
	if v := os.Getenv("SHIELD_SERVER_URL"); v == "" {
		t.Fatal("SHIELD_SERVER_URL must be set for acceptence tests")
	}

	if v := os.Getenv("SHIELD_USERNAME"); v == "" {
		t.Fatal("SHIELD_USERNAME must be set for acceptence tests")
	}
	if v := os.Getenv("SHIELD_PASSWORD"); v == "" {
		t.Fatal("SHIELD_PASSWORD must be set for acceptence tests")
	}
}
