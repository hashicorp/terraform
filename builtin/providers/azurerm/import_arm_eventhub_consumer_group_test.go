package azurerm

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMEventHubConsumerGroup_importBasic(t *testing.T) {
	resourceName := "azurerm_eventhub_consumer_group.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubConsumerGroup_basic, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubConsumerGroupDestroy,
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

func TestAccAzureRMEventHubConsumerGroup_importComplete(t *testing.T) {
	resourceName := "azurerm_eventhub_consumer_group.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubConsumerGroup_complete, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubConsumerGroupDestroy,
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
