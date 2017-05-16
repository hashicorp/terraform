package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCIPNetworkExchange_Basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccIPNetworkExchangeBasic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIPNetworkExchangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckIPNetworkExchangeExists,
			},
		},
	})
}

func testAccCheckIPNetworkExchangeExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPNetworkExchanges()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_network_exchange" {
			continue
		}

		input := compute.GetIPNetworkExchangeInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetIPNetworkExchange(&input); err != nil {
			return fmt.Errorf("Error retrieving state of ip network exchange %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckIPNetworkExchangeDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPNetworkExchanges()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_network_exchange" {
			continue
		}

		input := compute.GetIPNetworkExchangeInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetIPNetworkExchange(&input); err == nil {
			return fmt.Errorf("IPNetworkExchange %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

var testAccIPNetworkExchangeBasic = `
resource "opc_compute_ip_network_exchange" "test" {
  name = "test_ip_network_exchange-%d"
  description = "test ip network exchange"
}
`
