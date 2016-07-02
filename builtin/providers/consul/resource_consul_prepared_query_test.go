package consul

import (
	"fmt"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccConsulPreparedQuery_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConsulPreparedQueryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccConsulPreparedQueryConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulPreparedQueryExists(),
					testAccCheckConsulPreparedQueryAttrValue("name", "foo"),
					testAccCheckConsulPreparedQueryAttrValue("token", "bar"),
					testAccCheckConsulPreparedQueryAttrValue("service", "redis"),
					testAccCheckConsulPreparedQueryAttrValue("near", "_agent"),
					testAccCheckConsulPreparedQueryAttrValue("tags.#", "1"),
					testAccCheckConsulPreparedQueryAttrValue("only_passing", "true"),
					testAccCheckConsulPreparedQueryAttrValue("failover_nearest_n", "3"),
					testAccCheckConsulPreparedQueryAttrValue("failover_datacenters.#", "2"),
					testAccCheckConsulPreparedQueryAttrValue("dns_ttl", "8m"),
				),
			},
			resource.TestStep{
				Config: testAccConsulPreparedQueryConfigUpdate1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulPreparedQueryExists(),
					testAccCheckConsulPreparedQueryAttrValue("name", "baz"),
					testAccCheckConsulPreparedQueryAttrValue("token", "zip"),
					testAccCheckConsulPreparedQueryAttrValue("service", "memcached"),
					testAccCheckConsulPreparedQueryAttrValue("near", "node1"),
					testAccCheckConsulPreparedQueryAttrValue("tags.#", "2"),
					testAccCheckConsulPreparedQueryAttrValue("only_passing", "false"),
					testAccCheckConsulPreparedQueryAttrValue("failover_nearest_n", "2"),
					testAccCheckConsulPreparedQueryAttrValue("failover_datacenters.#", "1"),
					testAccCheckConsulPreparedQueryAttrValue("dns_ttl", "16m"),
				),
			},
			resource.TestStep{
				Config: testAccConsulPreparedQueryConfigUpdate2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulPreparedQueryExists(),
					testAccCheckConsulPreparedQueryAttrValue("token", ""),
					testAccCheckConsulPreparedQueryAttrValue("near", ""),
					testAccCheckConsulPreparedQueryAttrValue("tags.#", "0"),
					testAccCheckConsulPreparedQueryAttrValue("failover_nearest_n", "0"),
					testAccCheckConsulPreparedQueryAttrValue("failover_datacenters.#", "0"),
					testAccCheckConsulPreparedQueryAttrValue("dns_ttl", ""),
				),
			},
		},
	})
}

func checkPreparedQueryExists(s *terraform.State) bool {
	rn, ok := s.RootModule().Resources["consul_prepared_query.foo"]
	if !ok {
		return false
	}
	id := rn.Primary.ID

	client := testAccProvider.Meta().(*consulapi.Client).PreparedQuery()
	opts := &consulapi.QueryOptions{Datacenter: "dc1"}
	pq, _, err := client.Get(id, opts)
	return err == nil && pq != nil
}

func testAccCheckConsulPreparedQueryDestroy(s *terraform.State) error {
	if checkPreparedQueryExists(s) {
		return fmt.Errorf("Prepared query 'foo' still exists")
	}
	return nil
}

func testAccCheckConsulPreparedQueryExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !checkPreparedQueryExists(s) {
			return fmt.Errorf("Prepared query 'foo' does not exist")
		}
		return nil
	}
}

func testAccCheckConsulPreparedQueryAttrValue(attr, val string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rn, ok := s.RootModule().Resources["consul_prepared_query.foo"]
		if !ok {
			return fmt.Errorf("Resource not found")
		}
		out, ok := rn.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("Attribute '%s' not found: %#v", attr, rn.Primary.Attributes)
		}
		if out != val {
			return fmt.Errorf("Attribute '%s' value '%s' != '%s'", attr, out, val)
		}
		return nil
	}
}

const testAccConsulPreparedQueryConfig = `
resource "consul_prepared_query" "foo" {
	name = "foo"
	token = "bar"
	service = "redis"
	tags = ["prod"]
	near = "_agent"
	only_passing = true
	failover_nearest_n = 3
	failover_datacenters = ["dc1", "dc2"]
	dns_ttl = "8m"
}
`

const testAccConsulPreparedQueryConfigUpdate1 = `
resource "consul_prepared_query" "foo" {
	name = "baz"
	token = "zip"
	service = "memcached"
	tags = ["prod","sup"]
	near = "node1"
	only_passing = false
	failover_nearest_n = 2
	failover_datacenters = ["dc2"]
	dns_ttl = "16m"
}
`

const testAccConsulPreparedQueryConfigUpdate2 = `
resource "consul_prepared_query" "foo" {
	name = "baz"
	service = "memcached"
}
`
