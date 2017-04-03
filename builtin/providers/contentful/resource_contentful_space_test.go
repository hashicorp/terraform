package contentful

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	contentful "github.com/tolgaakyuz/contentful-go"
)

func TestAccContentfulSpace_Basic(t *testing.T) {
	t.Skip()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContentfulSpaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContentfulSpaceConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_space.myspace", "name", "TF Acc Test Space"),
			},
			resource.TestStep{
				Config: testAccContentfulSpaceUpdateConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_space.myspace", "name", "TF Acc Test Changed Space"),
			},
		},
	})
}

func testAccCheckContentfulSpaceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*contentful.Contentful)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contentful_space" {
			continue
		}

		space, err := client.Spaces.Get(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Space %s still exists after destroy", space.Sys.ID)
		}
	}

	return nil
}

var testAccContentfulSpaceConfig = `
resource "contentful_space" "myspace" {
  name = "TF Acc Test Space"
}
`

var testAccContentfulSpaceUpdateConfig = `
resource "contentful_space" "myspace" {
  name = "TF Acc Test Changed Space"
}
`
