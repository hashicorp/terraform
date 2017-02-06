package consul

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataConsulAgentSelf_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataConsulAgentSelfConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceValue("data.consul_agent_self.read", "bootstrap", "false"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "datacenter", "dc1"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "id", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "name", "<any>"),
					testAccCheckDataSourceValue("data.consul_agent_self.read", "server", "true"),
				),
			},
		},
	})
}

func testAccCheckDataSourceValue(n, attr, val string) resource.TestCheckFunc {
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

const testAccDataConsulAgentSelfConfig = `
data "consul_agent_self" "read" {
}
`
