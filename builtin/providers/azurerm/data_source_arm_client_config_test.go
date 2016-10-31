package azurerm

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMClientConfig_basic(t *testing.T) {
	clientId := os.Getenv("ARM_CLIENT_ID")
	tenantId := os.Getenv("ARM_TENANT_ID")
	subscriptionId := os.Getenv("ARM_SUBSCRIPTION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckArmClientConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAzureRMClientConfigAttr("data.azurerm_client_config.current", "client_id", clientId),
					testAzureRMClientConfigAttr("data.azurerm_client_config.current", "tenant_id", tenantId),
					testAzureRMClientConfigAttr("data.azurerm_client_config.current", "subscription_id", subscriptionId),
				),
			},
		},
	})
}

// Wraps resource.TestCheckResourceAttr to prevent leaking values to console
// in case of mismatch
func testAzureRMClientConfigAttr(name, key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		err := resource.TestCheckResourceAttr(name, key, value)(s)
		if err != nil {
			// return fmt.Errorf("%s: Attribute '%s', failed check (values hidden)", name, key)
			return err
		}

		return nil
	}
}

const testAccCheckArmClientConfig_basic = `
data "azurerm_client_config" "current" { }
`
