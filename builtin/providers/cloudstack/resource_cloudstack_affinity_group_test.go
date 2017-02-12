package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackAffinityGroup_basic(t *testing.T) {
	var affinityGroup cloudstack.AffinityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackAffinityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackAffinityGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackAffinityGroupExists("cloudstack_affinity_group.foo", &affinityGroup),
					testAccCheckCloudStackAffinityGroupAttributes(&affinityGroup),
				),
			},
		},
	})
}

func testAccCheckCloudStackAffinityGroupExists(
	n string, affinityGroup *cloudstack.AffinityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No affinity group ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		ag, _, err := cs.AffinityGroup.GetAffinityGroupByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if ag.Id != rs.Primary.ID {
			return fmt.Errorf("Affinity group not found")
		}

		*affinityGroup = *ag

		return nil
	}
}

func testAccCheckCloudStackAffinityGroupAttributes(
	affinityGroup *cloudstack.AffinityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if affinityGroup.Name != "terraform-affinity-group" {
			return fmt.Errorf("Bad name: %s", affinityGroup.Name)
		}

		if affinityGroup.Description != "terraform-affinity-group" {
			return fmt.Errorf("Bad description: %s", affinityGroup.Description)
		}

		if affinityGroup.Type != "host anti-affinity" {
			return fmt.Errorf("Bad type: %s", affinityGroup.Type)
		}

		return nil
	}
}

func testAccCheckCloudStackAffinityGroupDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_affinity_group" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No affinity group ID is set")
		}

		_, _, err := cs.AffinityGroup.GetAffinityGroupByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Affinity group %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackAffinityGroup = fmt.Sprintf(`
resource "cloudstack_affinity_group" "foo" {
	name = "terraform-affinity-group"
	type = "host anti-affinity"
}`)
