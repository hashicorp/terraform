package rabbitmq

import (
	"fmt"
	"strings"
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccExchange(t *testing.T) {
	var exchangeInfo rabbithole.ExchangeInfo
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccExchangeCheckDestroy(&exchangeInfo),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccExchangeConfig_basic,
				Check: testAccExchangeCheck(
					"rabbitmq_exchange.test", &exchangeInfo,
				),
			},
		},
	})
}

func testAccExchangeCheck(rn string, exchangeInfo *rabbithole.ExchangeInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("exchange id not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		exchParts := strings.Split(rs.Primary.ID, "@")

		exchanges, err := rmqc.ListExchangesIn(exchParts[1])
		if err != nil {
			return fmt.Errorf("Error retrieving exchange: %s", err)
		}

		for _, exchange := range exchanges {
			if exchange.Name == exchParts[0] && exchange.Vhost == exchParts[1] {
				exchangeInfo = &exchange
				return nil
			}
		}

		return fmt.Errorf("Unable to find exchange %s", rn)
	}
}

func testAccExchangeCheckDestroy(exchangeInfo *rabbithole.ExchangeInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rmqc := testAccProvider.Meta().(*rabbithole.Client)

		exchanges, err := rmqc.ListExchangesIn(exchangeInfo.Vhost)
		if err != nil {
			return fmt.Errorf("Error retrieving exchange: %s", err)
		}

		for _, exchange := range exchanges {
			if exchange.Name == exchangeInfo.Name && exchange.Vhost == exchangeInfo.Vhost {
				return fmt.Errorf("Exchange %s@%s still exist", exchangeInfo.Name, exchangeInfo.Vhost)
			}
		}

		return nil
	}
}

const testAccExchangeConfig_basic = `
resource "rabbitmq_vhost" "test" {
    name = "test"
}

resource "rabbitmq_permissions" "guest" {
    user = "guest"
    vhost = "${rabbitmq_vhost.test.name}"
    permissions {
        configure = ".*"
        write = ".*"
        read = ".*"
    }
}

resource "rabbitmq_exchange" "test" {
    name = "test"
    vhost = "${rabbitmq_permissions.guest.vhost}"
    settings {
        type = "fanout"
        durable = false
        auto_delete = true
    }
}`
