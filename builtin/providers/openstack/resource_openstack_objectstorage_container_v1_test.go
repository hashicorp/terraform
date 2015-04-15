package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/rackspace/gophercloud/openstack/objectstorage/v1/containers"
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
					resource.TestCheckResourceAttr("openstack_objectstorage_container_v1.container_1", "name", "tf-test-container"),
					resource.TestCheckResourceAttr("openstack_objectstorage_container_v1.container_1", "content_type", "application/json"),
				),
			},
			resource.TestStep{
				Config: testAccObjectStorageV1Container_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_objectstorage_container_v1.container_1", "content_type", "text/plain"),
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

var testAccObjectStorageV1Container_basic = fmt.Sprintf(`
	resource "openstack_objectstorage_container_v1" "container_1" {
		region = "%s"
		name = "tf-test-container"
		metadata {
			test = "true"
		}
		content_type = "application/json"
	}`,
	OS_REGION_NAME)

var testAccObjectStorageV1Container_update = fmt.Sprintf(`
	resource "openstack_objectstorage_container_v1" "container_1" {
		region = "%s"
		name = "tf-test-container"
		metadata {
			test = "true"
		}
		content_type = "text/plain"
	}`,
	OS_REGION_NAME)
