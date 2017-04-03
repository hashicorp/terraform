package ibmcloud

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccIBMCloudInfraVlan_Basic(t *testing.T) {

	createStepVlanName := acctest.RandString(16)
	updateStepVlanName := acctest.RandString(16)
	createConfig := fmt.Sprintf(testAccCheckIBMCloudInfraVlanConfigBasic, createStepVlanName)
	updateConfig := fmt.Sprintf(testAccCheckIBMCloudInfraVlanConfigBasic, updateStepVlanName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_vlan.test_vlan", "name", createStepVlanName),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_vlan.test_vlan", "datacenter", "lon02"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_vlan.test_vlan", "type", "PUBLIC"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_vlan.test_vlan", "softlayer_managed", "false"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_vlan.test_vlan", "router_hostname", "fcr01a.lon02"),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_vlan.test_vlan", "subnet_size", "8"),
				),
			},

			resource.TestStep{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_vlan.test_vlan", "name", updateStepVlanName),
				),
			},
		},
	})
}

const testAccCheckIBMCloudInfraVlanConfigBasic = `
resource "ibmcloud_infra_vlan" "test_vlan" {
   name = "%s"
   datacenter = "lon02"
   type = "PUBLIC"
   subnet_size = 8
   router_hostname = "fcr01a.lon02"
}`
