package google

import (
	"fmt"
	"testing"

	compute "google.golang.org/api/compute/v1"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeImage_resolveImage(t *testing.T) {
	var image compute.Image
	rand := acctest.RandString(10)
	name := fmt.Sprintf("test-image-%s", rand)
	fam := fmt.Sprintf("test-image-family-%s", rand)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeImage_resolving(name, fam),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeImageExists(
						"google_compute_image.foobar", &image),
					testAccCheckComputeImageResolution("google_compute_image.foobar"),
				),
			},
		},
	})
}

func testAccCheckComputeImageResolution(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		project := config.Project

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		if rs.Primary.Attributes["name"] == "" {
			return fmt.Errorf("No image name is set")
		}
		if rs.Primary.Attributes["family"] == "" {
			return fmt.Errorf("No image family is set")
		}
		if rs.Primary.Attributes["self_link"] == "" {
			return fmt.Errorf("No self_link is set")
		}

		name := rs.Primary.Attributes["name"]
		family := rs.Primary.Attributes["family"]
		link := rs.Primary.Attributes["self_link"]

		images := map[string]string{
			"family/debian-8":                                               "projects/debian-cloud/global/images/family/debian-8",
			"projects/debian-cloud/global/images/debian-8-jessie-v20170110": "projects/debian-cloud/global/images/debian-8-jessie-v20170110",
			"debian-8":                                                                                            "projects/debian-cloud/global/images/family/debian-8",
			"debian-8-jessie-v20170110":                                                                           "projects/debian-cloud/global/images/debian-8-jessie-v20170110",
			"https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-8-jessie-v20170110": "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-8-jessie-v20170110",

			"global/images/" + name:          "global/images/" + name,
			"global/images/family/" + family: "global/images/family/" + family,
			name:                   "global/images/" + name,
			family:                 "global/images/family/" + family,
			"family/" + family:     "global/images/family/" + family,
			project + "/" + name:   "projects/" + project + "/global/images/" + name,
			project + "/" + family: "projects/" + project + "/global/images/family/" + family,
			link: link,
		}

		for input, expectation := range images {
			result, err := resolveImage(config, input)
			if err != nil {
				return fmt.Errorf("Error resolving input %s to image: %+v\n", input, err)
			}
			if result != expectation {
				return fmt.Errorf("Expected input '%s' to resolve to '%s', it resolved to '%s' instead.\n", input, expectation, result)
			}
		}
		return nil
	}
}

func testAccComputeImage_resolving(name, family string) string {
	return fmt.Sprintf(`
resource "google_compute_disk" "foobar" {
	name = "%s"
	zone = "us-central1-a"
	image = "debian-8-jessie-v20160803"
}
resource "google_compute_image" "foobar" {
	name = "%s"
	family = "%s"
	source_disk = "${google_compute_disk.foobar.self_link}"
}
`, name, name, family)
}
