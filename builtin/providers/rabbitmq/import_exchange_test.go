package rabbitmq

import (
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccExchange_importBasic(t *testing.T) {
	resourceName := "rabbitmq_exchange.test"
	var exchangeInfo rabbithole.ExchangeInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccExchangeCheckDestroy(&exchangeInfo),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccExchangeConfig_basic,
				Check: testAccExchangeCheck(
					resourceName, &exchangeInfo,
				),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
