package brocadevtm

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"vtm": testAccProvider,
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
	if v := os.Getenv("VTM_USER"); v == "" {
		t.Fatal("VTM_USER must be set for acceptance tests")
	}

	if v := os.Getenv("VTM_PASSWORD"); v == "" {
		t.Fatal("VTM_PASSWORD must be set for acceptance tests")
	}

	if v := os.Getenv("VTM_SERVER"); v == "" {
		t.Fatal("VTM_SERVER must be set for acceptance tests")
	}
}
