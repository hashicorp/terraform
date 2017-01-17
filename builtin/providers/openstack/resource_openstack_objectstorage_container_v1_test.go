package openstack

import (
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccObjectStorageV1Container_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckObjectStorageV1ContainerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccObjectStorageV1Container_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"openstack_objectstorage_container_v1.container_1", "name", "container_1"),
					resource.TestCheckResourceAttr(
						"openstack_objectstorage_container_v1.container_1", "content_type", "application/json"),
				),
			},
			resource.TestStep{
				Config: testAccObjectStorageV1Container_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"openstack_objectstorage_container_v1.container_1", "content_type", "text/plain"),
				),
			},
		},
	})
}

func testAccCheckObjectStorageV1ContainerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	objectStorageClient, err := config.objectStorageV1Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack object storage client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_objectstorage_container_v1" {
			continue
		}

		_, err := containers.Get(objectStorageClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Container still exists")
		}
	}

	return nil
}

const testAccObjectStorageV1Container_basic = `
resource "openstack_objectstorage_container_v1" "container_1" {
  name = "container_1"
  metadata {
    test = "true"
  }
  content_type = "application/json"
}
`

const testAccObjectStorageV1Container_update = `
resource "openstack_objectstorage_container_v1" "container_1" {
  name = "container_1"
  metadata {
    test = "true"
  }
  content_type = "text/plain"
}
`
