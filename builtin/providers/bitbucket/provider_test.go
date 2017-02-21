package bitbucket

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
		"bitbucket": testAccProvider,
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
	if v := os.Getenv("BITBUCKET_USERNAME"); v == "" {
		t.Fatal("BITBUCKET_USERNAME must be set for acceptence tests")
	}
	if v := os.Getenv("BITBUCKET_PASSWORD"); v == "" {
		t.Fatal("BITBUCKET_PASSWORD must be set for acceptence tests")
	}
}
