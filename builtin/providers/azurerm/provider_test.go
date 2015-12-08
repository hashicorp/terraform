package azurerm

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

const (
	testAccSecurityGroupName = "terraform-security-group"
	testAccHostedServiceName = "terraform-testing-service"
)

// testAccStorageServiceName is used as the name for the Storage Service
// created in all storage-related tests.
// It is much more convenient to provide a Storage Service which
// has been created beforehand as the creation of one takes a lot
// and would greatly impede the multitude of tests which rely on one.
// NOTE: the storage container should be located in `West US`.
var testAccStorageServiceName = os.Getenv("AZURE_STORAGE")

const testAccStorageContainerName = "terraform-testing-container"

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"azurerm": testAccProvider,
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
	if v := os.Getenv("ARM_CREDENTIALS_FILE"); v == "" {
		subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
		clientID := os.Getenv("ARM_CLIENT_ID")
		clientSecret := os.Getenv("ARM_CLIENT_SECRET")
		tenantID := os.Getenv("ARM_TENANT_ID")

		if subscriptionID == "" || clientID == "" || clientSecret == "" || tenantID == "" {
			t.Fatal("Either ARM_CREDENTIALS_FILE or ARM_SUBSCRIPTION_ID, ARM_CLIENT_ID, " +
				"ARM_CLIENT_SECRET and ARM_TENANT_ID must be set for acceptance tests")
		}
	}

	if v := os.Getenv("AZURE_STORAGE"); v == "" {
		t.Fatal("AZURE_STORAGE must be set for acceptance tests")
	}
}
