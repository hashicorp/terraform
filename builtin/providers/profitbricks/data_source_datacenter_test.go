package profitbricks

import (
	"github.com/hashicorp/terraform/helper/resource"
	"regexp"
	"testing"
)

func TestAccDataSourceDatacenter_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{

				Config:      testAccDataSourceProfitBricksDataCenter_basic,
				ExpectError: regexp.MustCompile(`There are no datacenters that match the search criteria`),
			},
		},
	})

}

const testAccDataSourceProfitBricksDataCenter_basic = `
	data "profitbricks_datacenter" "dc_example" {
  	name = "test_name"
  	location = "us/las"
	}
	`
