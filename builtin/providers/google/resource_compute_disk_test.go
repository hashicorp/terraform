package google

import (
	"fmt"
	"testing"

	"code.google.com/p/google-api-go-client/compute/v1"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeDisk_basic(t *testing.T) {
	var disk compute.Disk

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeDisk_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeDiskExists(
						"google_compute_disk.foobar", &disk),
				),
			},
		},
	})
}

func testAccCheckComputeDiskDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_disk" {
			continue
		}

		_, err := config.clientCompute.Disks.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Disk still exists")
		}
	}

	return nil
}

func testAccCheckComputeDiskExists(n string, disk *compute.Disk) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Disks.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Disk not found")
		}

		*disk = *found

		return nil
	}
}

const testAccComputeDisk_basic = `
resource "google_compute_disk" "foobar" {
	name = "terraform-test"
	image = "debian-7-wheezy-v20140814"
	size = 50
	zone = "us-central1-a"
}`
