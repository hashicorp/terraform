package swift

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/ncw/swift"
	"strings"
)

func TestAccSwiftObject_Basic(t *testing.T) {
	var object swift.Object

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSwiftObjectCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:  testAccCheckSwiftObjectConfig_basic,
				Destroy: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSwiftObjectExists("swift_object.object-test-1", &object),
					resource.TestCheckResourceAttr(
						"swift_object.object-test-1", "name", "bar"),
					resource.TestCheckResourceAttr(
						"swift_object.object-test-1", "container_name", "foo"),
					resource.TestCheckResourceAttr(
						"swift_object.object-test-1", "contents", "hello, world!"),
				),
			},
		},
	})
}

func testAccCheckSwiftObjectCheckDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*swift.Connection)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "swift_object" {
			continue
		}

		id := strings.SplitN(rs.Primary.ID, "/", 2)
		containerName, objectName := id[0], id[1]

		// Try to find the guest
		_, err := c.ObjectGetBytes(containerName, objectName)

		if err == nil {
			return fmt.Errorf(
				"Swift object %s was not destroyed: %s",
				rs.Primary.ID, err.Error())
		}
	}

	return nil
}

func testAccCheckSwiftObjectExists(n string, object *swift.Object) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		err := testAccCheckSwiftContainerCheckDestroy(s)

		if err != nil {
			err = nil
		} else {
			err = fmt.Errorf(
				"swift provider test: testAccCheckSwiftObjectExists: object does not exist")
		}

		return err
	}
}

const testAccCheckSwiftObjectConfig_basic = `
resource "swift_container" "container-test-1" {
    name = "foo"
}
resource "swift_object" "object-test-1" {
    name = "bar"
    container_name = "${swift_container.container-test-1.id}"
    contents = "hello, world!"
}
`
