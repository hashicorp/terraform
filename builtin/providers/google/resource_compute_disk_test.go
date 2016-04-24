package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeDisk_basic(t *testing.T) {
	diskName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var disk compute.Disk

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeDisk_basic(diskName),
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

func testAccComputeDisk_basic(diskName string) string {
	return fmt.Sprintf(`
resource "google_compute_disk" "foobar" {
	name = "%s"
	image = "debian-7-wheezy-v20160301"
	size = 50
	type = "pd-ssd"
	zone = "us-central1-a"
}`, diskName)
}
