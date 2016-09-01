package cloudstack

import (
	"fmt"
	"strings"
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
				Config: testAccCloudStackAffinityGroupPair,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackAffinityGroupExists("terraform-test-affinity-group", &affinityGroup),
					testAccCheckCloudStackAffinityGroupAttributes(&affinityGroup),
					testAccCheckCloudStackAffinityGroupCreateAttributes("terraform-test-affinity-group"),
				),
			},
		},
	})
}

func testAccCheckCloudStackAffinityGroupExists(n string, affinityGroup *cloudstack.AffinityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No affinity group ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		p := cs.AffinityGroup.NewListAffinityGroupsParams()
		p.SetName(rs.Primary.ID)

		list, err := cs.AffinityGroup.ListAffinityGroups(p)
		if err != nil {
			return err
		}

		if list.Count != 1 || list.AffinityGroups[0].Name != rs.Primary.ID {
			return fmt.Errorf("Affinity group not found")
		}

		*affinityGroup = *list.AffinityGroups[0]

		return nil
	}
}

func testAccCheckCloudStackAffinityGroupAttributes(
	affinityGroup *cloudstack.AffinityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if affinityGroup.Type != CLOUDSTACK_AFFINITY_GROUP_TYPE {
			return fmt.Errorf("Affinity group: Attribute type expected %s, got %s",
				CLOUDSTACK_AFFINITY_GROUP_TYPE, affinityGroup.Type)
		}

		return nil
	}
}

func testAccCheckCloudStackAffinityGroupCreateAttributes(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		found := false

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "cloudstack_affinity_group" {
				continue
			}

			if rs.Primary.ID != name {
				continue
			}

			if !strings.Contains(rs.Primary.Attributes["description"], "terraform-test-description") {
				return fmt.Errorf(
					"Affiity group: Attribute description expected 'terraform-test-description' to be present, got %s",
					rs.Primary.Attributes["description"])
			}

			found = true
			break
		}

		if !found {
			return fmt.Errorf("Could not find affinity group %s", name)
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

		p := cs.AffinityGroup.NewListAffinityGroupsParams()
		p.SetName(rs.Primary.ID)

		r, err := cs.AffinityGroup.ListAffinityGroups(p)
		if err != nil {
			return err
		}

		for i := 0; i < r.Count; i++ {
			if r.AffinityGroups[i].Id == rs.Primary.ID {
				return fmt.Errorf("Affinity group %s still exists", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccCloudStackAffinityGroupPair = fmt.Sprintf(`
resource "cloudstack_affinity_group" "foo" {
  name = "terraform-test-affinty-group"
  type = "%s"
  description = "terraform-test-description"
}`, CLOUDSTACK_AFFINITY_GROUP_TYPE)
