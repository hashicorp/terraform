package opc

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCIPNetwork_Basic(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "opc_compute_ip_network.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resName, testAccOPCCheckIPNetworkDestroyed),
		Steps: []resource.TestStep{
			{
				Config: testAccOPCIPNetworkConfig_Basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(resName, testAccOPCCheckIPNetworkExists),
					resource.TestCheckResourceAttr(resName, "ip_address_prefix", "10.0.12.0/24"),
					resource.TestCheckResourceAttr(resName, "public_napt_enabled", "false"),
					resource.TestCheckResourceAttr(resName, "description", fmt.Sprintf("testing-desc-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("testing-ip-network-%d", rInt)),
					resource.TestMatchResourceAttr(resName, "uri", regexp.MustCompile("testing-ip-network")),
				),
			},
		},
	})
}

func TestAccOPCIPNetwork_Update(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "opc_compute_ip_network.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resName, testAccOPCCheckIPNetworkDestroyed),
		Steps: []resource.TestStep{
			{
				Config: testAccOPCIPNetworkConfig_Basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(resName, testAccOPCCheckIPNetworkExists),
					resource.TestCheckResourceAttr(resName, "ip_address_prefix", "10.0.12.0/24"),
					resource.TestCheckResourceAttr(resName, "public_napt_enabled", "false"),
					resource.TestCheckResourceAttr(resName, "description", fmt.Sprintf("testing-desc-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("testing-ip-network-%d", rInt)),
					resource.TestMatchResourceAttr(resName, "uri", regexp.MustCompile("testing-ip-network")),
				),
			},
			{
				Config: testAccOPCIPNetworkConfig_BasicUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(resName, testAccOPCCheckIPNetworkExists),
					resource.TestCheckResourceAttr(resName, "ip_address_prefix", "10.0.12.0/24"),
					resource.TestCheckResourceAttr(resName, "public_napt_enabled", "true"),
					resource.TestCheckResourceAttr(resName, "description", fmt.Sprintf("testing-desc-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("testing-ip-network-%d", rInt)),
				),
			},
		},
	})
}

func testAccOPCIPNetworkConfig_Basic(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_network" "test" {
  name = "testing-ip-network-%d"
  description = "testing-desc-%d"
  ip_address_prefix = "10.0.12.0/24"
}`, rInt, rInt)
}

func testAccOPCIPNetworkConfig_BasicUpdate(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_network" "test" {
  name = "testing-ip-network-%d"
  description = "testing-desc-%d"
  ip_address_prefix = "10.0.12.0/24"
  public_napt_enabled = true
}`, rInt, rInt)
}

func testAccOPCCheckIPNetworkExists(state *OPCResourceState) error {
	name := state.Attributes["name"]

	input := &compute.GetIPNetworkInput{
		Name: name,
	}

	if _, err := state.Client.IPNetworks().GetIPNetwork(input); err != nil {
		return fmt.Errorf("Error retrieving state of IP Network '%s': %v", name, err)
	}

	return nil
}

func testAccOPCCheckIPNetworkDestroyed(state *OPCResourceState) error {
	name := state.Attributes["name"]

	input := &compute.GetIPNetworkInput{
		Name: name,
	}

	if info, _ := state.Client.IPNetworks().GetIPNetwork(input); info != nil {
		return fmt.Errorf("IP Network '%s' still exists: %+v", name, info)
	}
	return nil
}
