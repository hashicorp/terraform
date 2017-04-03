package ibmcloud

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccIBMCloudInfraVlanDataSource_Basic(t *testing.T) {

	name := fmt.Sprintf("tfuat_vlan%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIBMCloudInfraVlanDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIBMCloudInfraResources("data.ibmcloud_infra_vlan.tfacc_vlan", "number",
						"ibmcloud_infra_vlan.test_vlan_private", "vlan_number"),
					resource.TestCheckResourceAttr("data.ibmcloud_infra_vlan.tfacc_vlan", "name", name),
					resource.TestMatchResourceAttr("data.ibmcloud_infra_vlan.tfacc_vlan", "id", regexp.MustCompile("^[0-9]+$")),
				),
			},
		},
	})
}

func testAccCheckIBMCloudInfraResources(srcResource, srcKey, tgtResource, tgtKey string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		sourceResource, ok := s.RootModule().Resources[srcResource]
		if !ok {
			return fmt.Errorf("Not found: %s", srcResource)
		}

		targetResource, ok := s.RootModule().Resources[tgtResource]
		if !ok {
			return fmt.Errorf("Not found: %s", tgtResource)
		}

		if sourceResource.Primary.Attributes[srcKey] != targetResource.Primary.Attributes[tgtKey] {
			return fmt.Errorf("Different values : Source : %s %s %s , Target : %s %s %s",
				srcResource, srcKey, sourceResource.Primary.Attributes[srcKey],
				tgtResource, tgtKey, targetResource.Primary.Attributes[tgtKey])
		}

		return nil
	}
}
func testAccCheckIBMCloudInfraVlanDataSourceConfig(name string) string {
	return fmt.Sprintf(`
    resource "ibmcloud_infra_vlan" "test_vlan_private" {
    name            = "%s"
    datacenter      = "dal06"
    type            = "PRIVATE"
    subnet_size     = 8
    
}
data "ibmcloud_infra_vlan" "tfacc_vlan" {
    number = "${ibmcloud_infra_vlan.test_vlan_private.vlan_number}"
    name = "${ibmcloud_infra_vlan.test_vlan_private.name}"
}`, name)
}
