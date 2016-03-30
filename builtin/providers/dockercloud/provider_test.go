package dockercloud

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
		"dockercloud": testAccProvider,
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
	if u := os.Getenv("DOCKERCLOUD_USER"); u == "" {
		t.Fatal("DOCKERCLOUD_USER must be set for acceptance tests")
	}
	if k := os.Getenv("DOCKERCLOUD_APIKEY"); k == "" {
		t.Fatal("DOCKERCLOUD_APIKEY must be set for acceptance tests")
	}
}
