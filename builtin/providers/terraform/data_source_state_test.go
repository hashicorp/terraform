package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestState_basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccState_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue(
						"data.terraform_remote_state.foo", "foo", "bar"),
				),
			},
		},
	})
}

func TestState_complexOutputs(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccState_complexOutputs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStateValue("terraform_remote_state.foo", "backend", "local"),
					testAccCheckStateValue("terraform_remote_state.foo", "config.path", "./test-fixtures/complex_outputs.tfstate"),
					testAccCheckStateValue("terraform_remote_state.foo", "computed_set.#", "2"),
					testAccCheckStateValue("terraform_remote_state.foo", `map.%`, "2"),
					testAccCheckStateValue("terraform_remote_state.foo", `map.key`, "test"),
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

		v := rs.Primary.Attributes[name]
		if v != value {
			return fmt.Errorf(
				"Value for %s is %s, not %s", name, v, value)
		}

		return nil
	}
}

const testAccState_basic = `
data "terraform_remote_state" "foo" {
	backend = "local"

	config {
		path = "./test-fixtures/basic.tfstate"
	}
}`

const testAccState_complexOutputs = `
resource "terraform_remote_state" "foo" {
	backend = "local"

	config {
		path = "./test-fixtures/complex_outputs.tfstate"
	}
}`
