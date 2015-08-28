package chef

import (
	"fmt"
	"reflect"
	"testing"

	chefc "github.com/go-chef/chef"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccNode_basic(t *testing.T) {
	var node chefc.Node

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccNodeCheckDestroy(&node),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNodeConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccNodeCheckExists("chef_node.test", &node),
					func(s *terraform.State) error {

						if expected := "terraform-acc-test-basic"; node.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, node.Name)
						}
						if expected := "terraform-acc-test-node-basic"; node.Environment != expected {
							return fmt.Errorf("wrong environment; expected %v, got %v", expected, node.Environment)
						}

						expectedRunList := []string{
							"recipe[terraform@1.0.0]",
							"recipe[consul]",
							"role[foo]",
						}
						if !reflect.DeepEqual(node.RunList, expectedRunList) {
							return fmt.Errorf("wrong runlist; expected %#v, got %#v", expectedRunList, node.RunList)
						}

						var expectedAttributes interface{}
						expectedAttributes = map[string]interface{}{
							"terraform_acc_test": true,
						}
						if !reflect.DeepEqual(node.AutomaticAttributes, expectedAttributes) {
							return fmt.Errorf("wrong automatic attributes; expected %#v, got %#v", expectedAttributes, node.AutomaticAttributes)
						}
						if !reflect.DeepEqual(node.NormalAttributes, expectedAttributes) {
							return fmt.Errorf("wrong normal attributes; expected %#v, got %#v", expectedAttributes, node.NormalAttributes)
						}
						if !reflect.DeepEqual(node.DefaultAttributes, expectedAttributes) {
							return fmt.Errorf("wrong default attributes; expected %#v, got %#v", expectedAttributes, node.DefaultAttributes)
						}
						if !reflect.DeepEqual(node.OverrideAttributes, expectedAttributes) {
							return fmt.Errorf("wrong override attributes; expected %#v, got %#v", expectedAttributes, node.OverrideAttributes)
						}

						return nil
					},
				),
			},
		},
	})
}

func testAccNodeCheckExists(rn string, node *chefc.Node) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("node id not set")
		}

		client := testAccProvider.Meta().(*chefc.Client)
		gotNode, err := client.Nodes.Get(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting node: %s", err)
		}

		*node = gotNode

		return nil
	}
}

func testAccNodeCheckDestroy(node *chefc.Node) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*chefc.Client)
		_, err := client.Nodes.Get(node.Name)
		if err == nil {
			return fmt.Errorf("node still exists")
		}
		if _, ok := err.(*chefc.ErrorResponse); !ok {
			// A more specific check is tricky because Chef Server can return
			// a few different error codes in this case depending on which
			// part of its stack catches the error.
			return fmt.Errorf("got something other than an HTTP error (%v) when getting node", err)
		}

		return nil
	}
}

const testAccNodeConfig_basic = `
resource "chef_environment" "test" {
  name = "terraform-acc-test-node-basic"
}
resource "chef_node" "test" {
  name = "terraform-acc-test-basic"
  environment_name = "terraform-acc-test-node-basic"
  automatic_attributes_json = <<EOT
{
     "terraform_acc_test": true
}
EOT
  normal_attributes_json = <<EOT
{
     "terraform_acc_test": true
}
EOT
  default_attributes_json = <<EOT
{
     "terraform_acc_test": true
}
EOT
  override_attributes_json = <<EOT
{
     "terraform_acc_test": true
}
EOT
  run_list = ["terraform@1.0.0", "recipe[consul]", "role[foo]"]
}
`
