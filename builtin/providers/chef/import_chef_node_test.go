package chef

import (
	"testing"

	chefc "github.com/go-chef/chef"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccChefNode_importBasic(t *testing.T) {
	var node chefc.Node

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNodeConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccNodeCheckExists("chef_node.test", &node),
				),
			},
			resource.TestStep{
				ResourceName:            "chef_node.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"automatic_attributes_json", "default_attributes_json", "normal_attributes_json", "override_attributes_json"}, // Attributes are not imported currently
			},
		},
	})
}
