package swift

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/ncw/swift"
)

func TestAccSwiftContainer_Basic(t *testing.T) {
	var container swift.Container

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSwiftContainerCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:  testAccCheckSwiftContainerConfig_basic,
				Destroy: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSwiftContainerExists("swift_container.terraform-acceptance-test-1", &container),
					resource.TestCheckResourceAttr(
						"swift_container.terraform-acceptance-test-1", "name", "terraform-swift-test"),
					resource.TestCheckResourceAttr(
						"swift_container.terraform-acceptance-test-1", "read_access.0", "foo_user"),
					resource.TestCheckResourceAttr(
						"swift_container.terraform-acceptance-test-1", "write_access.0", "bar_user"),
				),
			},
		},
	})
}

func testAccCheckSwiftContainerCheckDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*swift.Connection)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "swift_container" {
			continue
		}

		containerName := rs.Primary.ID

		// Try to find the guest
		_, _, err := c.Container(containerName)

		if err == nil {
			return fmt.Errorf(
				"Swift container %s was not destroyed: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckSwiftContainerExists(n string, container *swift.Container) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No container name is set")
		}

		containerName := rs.Primary.ID

		c := testAccProvider.Meta().(*swift.Connection)
		containerObj, _, err := c.Container(containerName)
		if err != nil {
			return err
		}

		fmt.Printf("The container name is %s", containerObj.Name)

		if containerObj.Name != containerName {
			return fmt.Errorf("Container %s not found", containerName)
		}

		*container = containerObj

		return nil
	}
}

const testAccCheckSwiftContainerConfig_basic = `
resource "swift_container" "terraform-acceptance-test-1" {
    name = "terraform-swift-test"
    read_access = ["foo_user"]
    write_access = ["bar_user"]
}
`
