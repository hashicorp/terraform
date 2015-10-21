package kubernetes

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
		"kubernetes": testAccProvider,
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
	if v := os.Getenv("KUBERNETES_ENDPOINT"); v == "" {
		t.Fatal("KUBERNETES_ENDPOINT must be set for acceptance tests")
	}
	if v := os.Getenv("KUBERNETES_USERNAME"); v == "" {
		t.Fatal("KUBERNETES_USERNAME must be set for acceptance tests")
	}
	if v := os.Getenv("KUBERNETES_PASSWORD"); v == "" {
		t.Fatal("KUBERNETES_PASSWORD must be set for acceptance tests")
	}
}
