package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var testAccPublicCloudUserConfig = fmt.Sprintf(`
resource "ovh_publiccloud_user" "user" {
	project_id  = "%s"
  description = "my user for acceptance tests"
}
`, os.Getenv("OVH_PUBLIC_CLOUD"))

func TestAccPublicCloudUser_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCheckPublicCloudUserPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPublicCloudUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPublicCloudUserConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPublicCloudUserExists("ovh_publiccloud_user.user", t),
					testAccCheckPublicCloudUserOpenRC("ovh_publiccloud_user.user", t),
				),
			},
		},
	})
}

func testAccCheckPublicCloudUserPreCheck(t *testing.T) {
	testAccPreCheck(t)
	testAccCheckPublicCloudExists(t)
}

func testAccCheckPublicCloudUserExists(n string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.Attributes["project_id"] == "" {
			return fmt.Errorf("No Project ID is set")
		}

		return publicCloudUserExists(rs.Primary.Attributes["project_id"], rs.Primary.ID, config.OVHClient)
	}
}

func testAccCheckPublicCloudUserOpenRC(n string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.Attributes["openstack_rc.OS_AUTH_URL"] == "" {
			return fmt.Errorf("No openstack_rc.OS_AUTH_URL is set")
		}

		if rs.Primary.Attributes["openstack_rc.OS_TENANT_ID"] == "" {
			return fmt.Errorf("No openstack_rc.OS_TENANT_ID is set")
		}

		if rs.Primary.Attributes["openstack_rc.OS_TENANT_NAME"] == "" {
			return fmt.Errorf("No openstack_rc.OS_TENANT_NAME is set")
		}

		if rs.Primary.Attributes["openstack_rc.OS_USERNAME"] == "" {
			return fmt.Errorf("No openstack_rc.OS_USERNAME is set")
		}

		return nil
	}
}

func testAccCheckPublicCloudUserDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ovh_publiccloud_user" {
			continue
		}

		err := publicCloudUserExists(rs.Primary.Attributes["project_id"], rs.Primary.ID, config.OVHClient)
		if err == nil {
			return fmt.Errorf("VRack > Public Cloud User still exists")
		}

	}
	return nil
}
