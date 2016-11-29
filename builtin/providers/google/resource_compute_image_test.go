package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeImage_basic(t *testing.T) {
	var image compute.Image

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeImageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeImage_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeImageExists(
						"google_compute_image.foobar", &image),
				),
			},
		},
	})
}

func TestAccComputeImage_basedondisk(t *testing.T) {
	var image compute.Image

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeImageDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeImage_basedondisk,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeImageExists(
						"google_compute_image.foobar", &image),
				),
			},
		},
	})
}

func testAccCheckComputeImageDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_image" {
			continue
		}

		_, err := config.clientCompute.Images.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Image still exists")
		}
	}

	return nil
}

func testAccCheckComputeImageExists(n string, image *compute.Image) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Images.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Image not found")
		}

		*image = *found

		return nil
	}
}

var testAccComputeImage_basic = fmt.Sprintf(`
resource "google_compute_image" "foobar" {
	name = "image-test-%s"
	raw_disk {
	  source = "https://storage.googleapis.com/bosh-cpi-artifacts/bosh-stemcell-3262.4-google-kvm-ubuntu-trusty-go_agent-raw.tar.gz"
	}
	create_timeout = 5
}`, acctest.RandString(10))

var testAccComputeImage_basedondisk = fmt.Sprintf(`
resource "google_compute_disk" "foobar" {
	name = "disk-test-%s"
	zone = "us-central1-a"
	image = "debian-8-jessie-v20160803"
}
resource "google_compute_image" "foobar" {
	name = "image-test-%s"
	source_disk = "${google_compute_disk.foobar.self_link}"
}`, acctest.RandString(10), acctest.RandString(10))
