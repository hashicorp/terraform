package contentful

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
		"contentful": testAccProvider,
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
	var cmaToken, organizationID string
	if cmaToken = os.Getenv("CONTENTFUL_MANAGEMENT_TOKEN"); cmaToken == "" {
		t.Fatal("CONTENTFUL_MANAGEMENT_TOKEN must set with a valid Contentful Content Management API Token for acceptance tests")
	}
	if organizationID = os.Getenv("CONTENTFUL_ORGANIZATION_ID"); organizationID == "" {
		t.Fatal("CONTENTFUL_ORGANIZATION_ID must set with a valid Contentful Organization ID for acceptance tests")
	}
}
