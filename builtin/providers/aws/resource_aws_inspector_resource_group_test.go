package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSInspectorResourceGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSInspectorResourceGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSInspectorResourceGroupExists("aws_inspector_resource_group.foo"),
				),
			},
			resource.TestStep{
				Config: testAccCheckAWSInspectorResourceGroupModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSInspectorTargetExists("aws_inspector_resource_group.foo"),
				),
			},
		},
	})
}

func testAccCheckAWSInspectorResourceGroupExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		return nil
	}
}

var testAccAWSInspectorResourceGroup = `
resource "aws_inspector_resource_group" "foo" {
	tags {
	  Name  = "foo"
  }
}`

var testAccCheckAWSInspectorResourceGroupModified = `
resource "aws_inspector_resource_group" "foo" {
	tags {
	  Name  = "bar"
  }
}`
