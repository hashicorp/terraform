package scaleway

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccScalewayDataSourceImage_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckScalewayImageConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageID("data.scaleway_image.ubuntu"),
					resource.TestCheckResourceAttr("data.scaleway_image.ubuntu", "architecture", "arm"),
					resource.TestCheckResourceAttr("data.scaleway_image.ubuntu", "public", "true"),
					resource.TestCheckResourceAttrSet("data.scaleway_image.ubuntu", "organization"),
					resource.TestCheckResourceAttrSet("data.scaleway_image.ubuntu", "creation_date"),
				),
			},
		},
	})
}

func TestAccScalewayDataSourceImage_Filtered(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckScalewayImageFilterConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImageID("data.scaleway_image.ubuntu"),
					resource.TestCheckResourceAttr("data.scaleway_image.ubuntu", "name", "Ubuntu Precise (12.04)"),
					resource.TestCheckResourceAttr("data.scaleway_image.ubuntu", "architecture", "arm"),
					resource.TestCheckResourceAttr("data.scaleway_image.ubuntu", "public", "true"),
					resource.TestCheckResourceAttrSet("data.scaleway_image.ubuntu", "organization"),
					resource.TestCheckResourceAttrSet("data.scaleway_image.ubuntu", "creation_date"),
				),
			},
		},
	})
}

func testAccCheckImageID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find image data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("image data source ID not set")
		}

		scaleway := testAccProvider.Meta().(*Client).scaleway
		_, err := scaleway.GetImage(rs.Primary.ID)

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccCheckScalewayImageConfig = `
data "scaleway_image" "ubuntu" {
  name = "Ubuntu Precise"
  architecture = "arm"
}
`

const testAccCheckScalewayImageFilterConfig = `
data "scaleway_image" "ubuntu" {
  name_filter = "Precise"
  architecture = "arm"
}
`
