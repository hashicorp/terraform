package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccState_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccState_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue(
						"terraform_remote_state.foo", "foo", "bar"),
				),
			},
		},
	})
}

func testAccCheckStateValue(id, name, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		v := rs.Primary.Attributes["output."+name]
		if v != value {
			return fmt.Errorf(
				"Value for %s is %s, not %s", name, v, value)
		}

		return nil
	}
}

const testAccState_basic = `
resource "terraform_remote_state" "foo" {
	backend = "_local"

	config {
		path = "./test-fixtures/basic.tfstate"
	}
}`
