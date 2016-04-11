package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackDisk_basic(t *testing.T) {
	var disk cloudstack.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackDisk_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackDiskExists(
						"cloudstack_disk.foo", &disk),
					testAccCheckCloudStackDiskAttributes(&disk),
				),
			},
		},
	})
}

func TestAccCloudStackDisk_device(t *testing.T) {
	var disk cloudstack.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackDisk_device,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackDiskExists(
						"cloudstack_disk.foo", &disk),
					testAccCheckCloudStackDiskAttributes(&disk),
					resource.TestCheckResourceAttr(
						"cloudstack_disk.foo", "device", "/dev/xvde"),
				),
			},
		},
	})
}

func TestAccCloudStackDisk_update(t *testing.T) {
	var disk cloudstack.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackDisk_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackDiskExists(
						"cloudstack_disk.foo", &disk),
					testAccCheckCloudStackDiskAttributes(&disk),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackDisk_resize,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackDiskExists(
						"cloudstack_disk.foo", &disk),
					testAccCheckCloudStackDiskResized(&disk),
					resource.TestCheckResourceAttr(
						"cloudstack_disk.foo", "disk_offering", CLOUDSTACK_DISK_OFFERING_2),
				),
			},
		},
	})
}

func testAccCheckCloudStackDiskExists(
	n string, disk *cloudstack.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No disk ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		volume, _, err := cs.Volume.GetVolumeByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if volume.Id != rs.Primary.ID {
			return fmt.Errorf("Disk not found")
		}

		*disk = *volume

		return nil
	}
}

func testAccCheckCloudStackDiskAttributes(
	disk *cloudstack.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if disk.Name != "terraform-disk" {
			return fmt.Errorf("Bad name: %s", disk.Name)
		}

		if disk.Diskofferingname != CLOUDSTACK_DISK_OFFERING_1 {
			return fmt.Errorf("Bad disk offering: %s", disk.Diskofferingname)
		}

		return nil
	}
}

func testAccCheckCloudStackDiskResized(
	disk *cloudstack.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if disk.Diskofferingname != CLOUDSTACK_DISK_OFFERING_2 {
			return fmt.Errorf("Bad disk offering: %s", disk.Diskofferingname)
		}

		return nil
	}
}

func testAccCheckCloudStackDiskDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_disk" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No disk ID is set")
		}

		_, _, err := cs.Volume.GetVolumeByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Disk %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackDisk_basic = fmt.Sprintf(`
resource "cloudstack_disk" "foo" {
  name = "terraform-disk"
  attach = false
  disk_offering = "%s"
  zone = "%s"
}`,
	CLOUDSTACK_DISK_OFFERING_1,
	CLOUDSTACK_ZONE)

var testAccCloudStackDisk_device = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_disk" "foo" {
  name = "terraform-disk"
  attach = true
  device = "/dev/xvde"
  disk_offering = "%s"
  virtual_machine = "${cloudstack_instance.foobar.name}"
  zone = "${cloudstack_instance.foobar.zone}"
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_DISK_OFFERING_1)

var testAccCloudStackDisk_update = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_disk" "foo" {
  name = "terraform-disk"
  attach = true
  disk_offering = "%s"
  virtual_machine = "${cloudstack_instance.foobar.name}"
  zone = "${cloudstack_instance.foobar.zone}"
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_DISK_OFFERING_1)

var testAccCloudStackDisk_resize = fmt.Sprintf(`
resource "cloudstack_instance" "foobar" {
  name = "terraform-test"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_disk" "foo" {
  name = "terraform-disk"
  attach = true
  disk_offering = "%s"
  virtual_machine = "${cloudstack_instance.foobar.name}"
  zone = "${cloudstack_instance.foobar.zone}"
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_DISK_OFFERING_2)
