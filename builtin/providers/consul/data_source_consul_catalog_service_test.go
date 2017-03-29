package consul

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataConsulCatalogService_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataConsulCatalogServiceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "datacenter", "dc1"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.#", "1"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.address", "<all>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.create_index", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.enable_tag_override", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.id", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.modify_index", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.name", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.node_address", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.node_id", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.node_meta.%", "0"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.node_name", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.port", "<any>"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.tagged_addresses.%", "2"),
					testAccCheckDataSourceValue("data.consul_catalog_service.read", "service.0.tags.#", "0"),
				),
			},
		},
	})
}

const testAccDataConsulCatalogServiceConfig = `
data "consul_catalog_service" "read" {
  query_options {
    allow_stale = true
    require_consistent = false
    token = ""
    wait_index = 0
    wait_time = "1m"
  }

  name = "consul"
}
`
