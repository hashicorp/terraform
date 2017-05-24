package packet

import (
	"os"
	"testing"

	"github.com/r3labs/terraform/helper/schema"
	"github.com/r3labs/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"packet": testAccProvider,
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
	if v := os.Getenv("PACKET_AUTH_TOKEN"); v == "" {
		t.Fatal("PACKET_AUTH_TOKEN must be set for acceptance tests")
	}
}
