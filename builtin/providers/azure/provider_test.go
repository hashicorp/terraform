package azure

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
		"azure": testAccProvider,
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
	if v := os.Getenv("AZURE_SETTINGS_FILE"); v == "" {
		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
		certificate := os.Getenv("AZURE_CERTIFICATE")

		if subscriptionID == "" || certificate == "" {
			t.Fatal("either AZURE_SETTINGS_FILE, or AZURE_SUBSCRIPTION_ID " +
				"and AZURE_CERTIFICATE must be set for acceptance tests")
		}
	}

	if v := os.Getenv("AZURE_STORAGE"); v == "" {
		t.Fatal("AZURE_STORAGE must be set for acceptance tests")
	}
}
