package profitbricks

import (
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccDataSourceImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{

				Config: testAccDataSourceProfitBricksImage_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.profitbricks_image.img", "location", "us/las"),
					resource.TestCheckResourceAttr("data.profitbricks_image.img", "name", "Ubuntu-16.04-LTS-server-2017-05-01"),
					resource.TestCheckResourceAttr("data.profitbricks_image.img", "type", "HDD"),
				),
			},
		},
	})

}

const testAccDataSourceProfitBricksImage_basic = `
	data "profitbricks_image" "img" {
	  name = "Ubuntu"
	  type = "HDD"
	  version = "16"
	  location = "us/las"
	}
	`
