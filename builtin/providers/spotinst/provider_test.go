package spotinst

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
		"spotinst": testAccProvider,
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
	if v := os.Getenv("SPOTINST_EMAIL"); v == "" {
		t.Fatal("SPOTINST_EMAIL must be set for acceptance tests")
	}

	if v := os.Getenv("SPOTINST_PASSWORD"); v == "" {
		t.Fatal("SPOTINST_PASSWORD must be set for acceptance tests")
	}

	if v := os.Getenv("SPOTINST_CLIENT_ID"); v == "" {
		t.Fatal("SPOTINST_CLIENT_ID must be set for acceptance tests")
	}

	if v := os.Getenv("SPOTINST_CLIENT_SECRET"); v == "" {
		t.Fatal("SPOTINST_CLIENT_SECRET must be set for acceptance tests")
	}
}
