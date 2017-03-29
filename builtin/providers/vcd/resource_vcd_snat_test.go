package vcd

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	govcd "github.com/ukcloud/govcloudair"
)

func TestAccVcdSNAT_Basic(t *testing.T) {
	if v := os.Getenv("VCD_EXTERNAL_IP"); v == "" {
		t.Skip("Environment variable VCD_EXTERNAL_IP must be set to run SNAT tests")
		return
	}

	var e govcd.EdgeGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdSNATDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckVcdSnat_basic, os.Getenv("VCD_EDGE_GATWEWAY"), os.Getenv("VCD_EXTERNAL_IP")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdSNATExists("vcd_snat.bar", &e),
					resource.TestCheckResourceAttr(
						"vcd_snat.bar", "external_ip", os.Getenv("VCD_EXTERNAL_IP")),
					resource.TestCheckResourceAttr(
						"vcd_snat.bar", "internal_ip", "10.10.102.0/24"),
				),
			},
		},
	})
}

func testAccCheckVcdSNATExists(n string, gateway *govcd.EdgeGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		//return fmt.Errorf("Check this: %#v", rs.Primary)

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNAT ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		gatewayName := rs.Primary.Attributes["edge_gateway"]
		edgeGateway, err := conn.OrgVdc.FindEdgeGateway(gatewayName)

		if err != nil {
			return fmt.Errorf("Could not find edge gateway")
		}

		var found bool
		for _, v := range edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
			if v.RuleType == "SNAT" &&
				v.GatewayNatRule.OriginalIP == "10.10.102.0/24" &&
				v.GatewayNatRule.OriginalPort == "" &&
				v.GatewayNatRule.TranslatedIP == os.Getenv("VCD_EXTERNAL_IP") {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("SNAT rule was not found")
		}

		*gateway = edgeGateway

		return nil
	}
}

func testAccCheckVcdSNATDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*VCDClient)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_snat" {
			continue
		}

		gatewayName := rs.Primary.Attributes["edge_gateway"]
		edgeGateway, err := conn.OrgVdc.FindEdgeGateway(gatewayName)

		if err != nil {
			return fmt.Errorf("Could not find edge gateway")
		}

		var found bool
		for _, v := range edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
			if v.RuleType == "SNAT" &&
				v.GatewayNatRule.OriginalIP == "10.10.102.0/24" &&
				v.GatewayNatRule.OriginalPort == "" &&
				v.GatewayNatRule.TranslatedIP == os.Getenv("VCD_EXTERNAL_IP") {
				found = true
			}
		}

		if found {
			return fmt.Errorf("SNAT rule still exists.")
		}
	}

	return nil
}

const testAccCheckVcdSnat_basic = `
resource "vcd_snat" "bar" {
	edge_gateway = "%s"
	external_ip = "%s"
	internal_ip = "10.10.102.0/24"
}
`
