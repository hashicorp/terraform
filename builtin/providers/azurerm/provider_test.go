package azurerm

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProviderFactory
var testAccProvider = Provider().(*schema.Provider)

func init() {
	testAccProviders = map[string]terraform.ResourceProviderFactory{
		"azurerm": func() (terraform.ResourceProvider, error) {
			// The StopContext needs to be replaced if it was used in a test.

			// Reset the Context in the schema.Provider.
			testAccProvider.StopContextTestReset()

			// get the configured client and replace its copy of the Context
			m := testAccProvider.Meta()
			if m != nil {
				m.(*ArmClient).StopContext = testAccProvider.StopContext()
				testAccProvider.SetMeta(m)
			}
			return testAccProvider, nil
		},
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
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	clientID := os.Getenv("ARM_CLIENT_ID")
	clientSecret := os.Getenv("ARM_CLIENT_SECRET")
	tenantID := os.Getenv("ARM_TENANT_ID")

	if subscriptionID == "" || clientID == "" || clientSecret == "" || tenantID == "" {
		t.Fatal("ARM_SUBSCRIPTION_ID, ARM_CLIENT_ID, ARM_CLIENT_SECRET and ARM_TENANT_ID must be set for acceptance tests")
	}
}
