package profitbricks

import (
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccDataSourceDatacenter_matching(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{

				Config: testAccDataSourceProfitBricksDataCenter_matching,
			},
			{

				Config: testAccDataSourceProfitBricksDataCenter_matchingWithDataSource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.profitbricks_datacenter.foobar", "name", "test_name"),
					resource.TestCheckResourceAttr("data.profitbricks_datacenter.foobar", "location", "us/las"),
				),
			},
		},
	})

}

const testAccDataSourceProfitBricksDataCenter_matching = `
resource "profitbricks_datacenter" "foobar" {
    name       = "test_name"
    location = "us/las"
}
`

const testAccDataSourceProfitBricksDataCenter_matchingWithDataSource = `
resource "profitbricks_datacenter" "foobar" {
    name       = "test_name"
    location = "us/las"
}

data "profitbricks_datacenter" "foobar" {
    name = "${profitbricks_datacenter.foobar.name}"
    location = "us/las"
}`
