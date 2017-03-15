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
					testAccCheckConsulPreparedQueryAttrValue("stored_token", "pq-token"),
					testAccCheckConsulPreparedQueryAttrValue("service", "redis"),
					testAccCheckConsulPreparedQueryAttrValue("near", "_agent"),
					testAccCheckConsulPreparedQueryAttrValue("tags.#", "1"),
					testAccCheckConsulPreparedQueryAttrValue("only_passing", "true"),
					testAccCheckConsulPreparedQueryAttrValue("failover.0.nearest_n", "3"),
					testAccCheckConsulPreparedQueryAttrValue("failover.0.datacenters.#", "2"),
					testAccCheckConsulPreparedQueryAttrValue("template.0.type", "name_prefix_match"),
					testAccCheckConsulPreparedQueryAttrValue("template.0.regexp", "hello"),
					testAccCheckConsulPreparedQueryAttrValue("dns.0.ttl", "8m"),
				),
			},
			resource.TestStep{
				Config: testAccConsulPreparedQueryConfigUpdate1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulPreparedQueryExists(),
					testAccCheckConsulPreparedQueryAttrValue("name", "baz"),
					testAccCheckConsulPreparedQueryAttrValue("stored_token", "pq-token-updated"),
					testAccCheckConsulPreparedQueryAttrValue("service", "memcached"),
					testAccCheckConsulPreparedQueryAttrValue("near", "node1"),
					testAccCheckConsulPreparedQueryAttrValue("tags.#", "2"),
					testAccCheckConsulPreparedQueryAttrValue("only_passing", "false"),
					testAccCheckConsulPreparedQueryAttrValue("failover.0.nearest_n", "2"),
					testAccCheckConsulPreparedQueryAttrValue("failover.0.datacenters.#", "1"),
					testAccCheckConsulPreparedQueryAttrValue("template.0.regexp", "goodbye"),
					testAccCheckConsulPreparedQueryAttrValue("dns.0.ttl", "16m"),
				),
			},
			resource.TestStep{
				Config: testAccConsulPreparedQueryConfigUpdate2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulPreparedQueryExists(),
					testAccCheckConsulPreparedQueryAttrValue("stored_token", ""),
					testAccCheckConsulPreparedQueryAttrValue("near", ""),
					testAccCheckConsulPreparedQueryAttrValue("tags.#", "0"),
					testAccCheckConsulPreparedQueryAttrValue("failover.#", "0"),
					testAccCheckConsulPreparedQueryAttrValue("template.#", "0"),
					testAccCheckConsulPreparedQueryAttrValue("dns.#", "0"),
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
	token = "client-token"
	stored_token = "pq-token"
	service = "redis"
	tags = ["prod"]
	near = "_agent"
	only_passing = true

	failover {
		nearest_n = 3
		datacenters = ["dc1", "dc2"]
	}

	template {
		type = "name_prefix_match"
		regexp = "hello"
	}

	dns {
		ttl = "8m"
	}
}
`

const testAccConsulPreparedQueryConfigUpdate1 = `
resource "consul_prepared_query" "foo" {
	name = "baz"
	token = "client-token"
	stored_token = "pq-token-updated"
	service = "memcached"
	tags = ["prod","sup"]
	near = "node1"
	only_passing = false

	failover {
		nearest_n = 2
		datacenters = ["dc2"]
	}

	template {
		type = "name_prefix_match"
		regexp = "goodbye"
	}

	dns {
		ttl = "16m"
	}
}
`

const testAccConsulPreparedQueryConfigUpdate2 = `
resource "consul_prepared_query" "foo" {
	name = "baz"
	service = "memcached"
	token = "client-token"
}
`
