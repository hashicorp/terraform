package vcd

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	govcd "github.com/ukcloud/govcloudair"
)

func TestAccVcdFirewallRules_basic(t *testing.T) {

	var existingRules, fwRules govcd.EdgeGateway
	newConfig := createFirewallRulesConfigs(&existingRules)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: newConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdFirewallRulesExists("vcd_firewall_rules.bar", &fwRules),
					testAccCheckVcdFirewallRulesAttributes(&fwRules, &existingRules),
				),
			},
		},
	})

}

func testAccCheckVcdFirewallRulesExists(n string, gateway *govcd.EdgeGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		resp, err := conn.OrgVdc.FindEdgeGateway(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Edge Gateway does not exist.")
		}

		*gateway = resp

		return nil
	}
}

func testAccCheckVcdFirewallRulesAttributes(newRules, existingRules *govcd.EdgeGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if len(newRules.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule) != len(existingRules.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule)+1 {
			return fmt.Errorf("New firewall rule not added: %d != %d",
				len(newRules.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule),
				len(existingRules.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule)+1)
		}

		return nil
	}
}

func createFirewallRulesConfigs(existingRules *govcd.EdgeGateway) string {
	config := Config{
		User:            os.Getenv("VCD_USER"),
		Password:        os.Getenv("VCD_PASSWORD"),
		Org:             os.Getenv("VCD_ORG"),
		Href:            os.Getenv("VCD_URL"),
		VDC:             os.Getenv("VCD_VDC"),
		MaxRetryTimeout: 240,
	}
	conn, err := config.Client()
	if err != nil {
		return fmt.Sprintf(testAccCheckVcdFirewallRules_add, "", "")
	}
	edgeGateway, _ := conn.OrgVdc.FindEdgeGateway(os.Getenv("VCD_EDGE_GATWEWAY"))
	*existingRules = edgeGateway
	log.Printf("[DEBUG] Edge gateway: %#v", edgeGateway)
	firewallRules := *edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService
	return fmt.Sprintf(testAccCheckVcdFirewallRules_add, os.Getenv("VCD_EDGE_GATEWAY"), firewallRules.DefaultAction)
}

const testAccCheckVcdFirewallRules_add = `
resource "vcd_firewall_rules" "bar" {
  edge_gateway = "%s"
	default_action = "%s"

	rule {
		description = "Test rule"
		policy = "allow"
		protocol = "any"
		destination_port = "any"
		destination_ip = "any"
		source_port = "any"
		source_ip = "any"
	}
}
`
