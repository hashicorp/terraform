package vcd

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	govcd "github.com/ukcloud/govcloudair"
)

func TestAccVcdDNAT_Basic(t *testing.T) {
	if v := os.Getenv("VCD_EXTERNAL_IP"); v == "" {
		t.Skip("Environment variable VCD_EXTERNAL_IP must be set to run DNAT tests")
		return
	}

	var e govcd.EdgeGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdDNATDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckVcdDnat_basic, os.Getenv("VCD_EDGE_GATWEWAY"), os.Getenv("VCD_EXTERNAL_IP")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdDNATExists("vcd_dnat.bar", &e),
					resource.TestCheckResourceAttr(
						"vcd_dnat.bar", "external_ip", os.Getenv("VCD_EXTERNAL_IP")),
					resource.TestCheckResourceAttr(
						"vcd_dnat.bar", "port", "77"),
					resource.TestCheckResourceAttr(
						"vcd_dnat.bar", "internal_ip", "10.10.102.60"),
				),
			},
		},
	})
}

func testAccCheckVcdDNATExists(n string, gateway *govcd.EdgeGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DNAT ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		gatewayName := rs.Primary.Attributes["edge_gateway"]
		edgeGateway, err := conn.OrgVdc.FindEdgeGateway(gatewayName)

		if err != nil {
			return fmt.Errorf("Could not find edge gateway")
		}

		var found bool
		for _, v := range edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
			if v.RuleType == "DNAT" &&
				v.GatewayNatRule.OriginalIP == os.Getenv("VCD_EXTERNAL_IP") &&
				v.GatewayNatRule.OriginalPort == "77" &&
				v.GatewayNatRule.TranslatedIP == "10.10.102.60" {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("DNAT rule was not found")
		}

		*gateway = edgeGateway

		return nil
	}
}

func testAccCheckVcdDNATDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*VCDClient)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_dnat" {
			continue
		}

		gatewayName := rs.Primary.Attributes["edge_gateway"]
		edgeGateway, err := conn.OrgVdc.FindEdgeGateway(gatewayName)

		if err != nil {
			return fmt.Errorf("Could not find edge gateway")
		}

		var found bool
		for _, v := range edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
			if v.RuleType == "DNAT" &&
				v.GatewayNatRule.OriginalIP == os.Getenv("VCD_EXTERNAL_IP") &&
				v.GatewayNatRule.OriginalPort == "77" &&
				v.GatewayNatRule.TranslatedIP == "10.10.102.60" {
				found = true
			}
		}

		if found {
			return fmt.Errorf("DNAT rule still exists.")
		}
	}

	return nil
}

const testAccCheckVcdDnat_basic = `
resource "vcd_dnat" "bar" {
	edge_gateway = "%s"
	external_ip = "%s"
	port = 77
	internal_ip = "10.10.102.60"
}
`
