package icinga2

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
		"icinga2": testAccProvider,
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

	v := os.Getenv("ICINGA2_API_URL")
	if v == "" {
		t.Fatal("ICINGA2_API_URL must be set for acceptance tests")
	}

	v = os.Getenv("ICINGA2_API_USER")
	if v == "" {
		t.Fatal("ICINGA2_API_USER must be set for acceptance tests")
	}

	v = os.Getenv("ICINGA2_API_PASSWORD")
	if v == "" {
		t.Fatal("ICINGA2_API_PASSWORD must be set for acceptance tests")
	}

}
