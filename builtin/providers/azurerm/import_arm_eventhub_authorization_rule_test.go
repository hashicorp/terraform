package azurerm

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMEventHubAuthorizationRule_importListen(t *testing.T) {
	resourceName := "azurerm_eventhub_authorization_rule.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubAuthorizationRule_listen, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubAuthorizationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMEventHubAuthorizationRule_importSend(t *testing.T) {
	resourceName := "azurerm_eventhub_authorization_rule.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubAuthorizationRule_send, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubAuthorizationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMEventHubAuthorizationRule_importReadWrite(t *testing.T) {
	resourceName := "azurerm_eventhub_authorization_rule.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubAuthorizationRule_readwrite, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubAuthorizationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMEventHubAuthorizationRule_importManage(t *testing.T) {
	resourceName := "azurerm_eventhub_authorization_rule.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubAuthorizationRule_manage, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubAuthorizationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
