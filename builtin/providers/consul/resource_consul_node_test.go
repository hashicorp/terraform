package consul

import (
	"fmt"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccConsulNode_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() {},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConsulNodeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccConsulNodeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConsulNodeExists(),
					testAccCheckConsulNodeValue("consul_catalog_entry.foo", "address", "127.0.0.1"),
					testAccCheckConsulNodeValue("consul_catalog_entry.foo", "node", "foo"),
				),
			},
		},
	})
}

func testAccCheckConsulNodeDestroy(s *terraform.State) error {
	catalog := testAccProvider.Meta().(*consulapi.Client).Catalog()
	qOpts := consulapi.QueryOptions{}
	nodes, _, err := catalog.Nodes(&qOpts)
	if err != nil {
		return fmt.Errorf("Could not retrieve services: %#v", err)
	}
	for i := range nodes {
		if nodes[i].Node == "foo" {
			return fmt.Errorf("Node still exists: %#v", "foo")
		}
	}
	return nil
}

func testAccCheckConsulNodeExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		catalog := testAccProvider.Meta().(*consulapi.Client).Catalog()
		qOpts := consulapi.QueryOptions{}
		nodes, _, err := catalog.Nodes(&qOpts)
		if err != nil {
			return err
		}
		for i := range nodes {
			if nodes[i].Node == "foo" {
				return nil
			}
		}
		return fmt.Errorf("Service does not exist: %#v", "google")
	}
}

func testAccCheckConsulNodeValue(n, attr, val string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rn, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found")
		}
		out, ok := rn.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("Attribute '%s' not found: %#v", attr, rn.Primary.Attributes)
		}
		if val != "<any>" && out != val {
			return fmt.Errorf("Attribute '%s' value '%s' != '%s'", attr, out, val)
		}
		if val == "<any>" && out == "" {
			return fmt.Errorf("Attribute '%s' value '%s'", attr, out)
		}
		return nil
	}
}

const testAccConsulNodeConfig = `
resource "consul_catalog_entry" "foo" {
	address = "127.0.0.1"
	node = "foo"
}
`
