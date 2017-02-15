package consul

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataConsulCatalogServices_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataConsulCatalogServicesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceValue("data.consul_catalog_services.read", "datacenter", "dc1"),
					testAccCheckDataSourceValue("data.consul_catalog_services.read", "services.%", "1"),
					testAccCheckDataSourceValue("data.consul_catalog_services.read", "services.consul", ""),
				),
			},
		},
	})
}

const testAccDataConsulCatalogServicesConfig = `
data "consul_catalog_services" "read" {
  query_options {
    allow_stale = true
    require_consistent = false
    token = ""
    wait_index = 0
    wait_time = "1m"
  }
}
`
