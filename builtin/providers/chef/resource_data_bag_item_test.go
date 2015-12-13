package chef

import (
	"fmt"
	"reflect"
	"testing"

	chefc "github.com/go-chef/chef"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataBagItem_basic(t *testing.T) {
	var dataBagItemName string
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDataBagItemCheckDestroy(dataBagItemName),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataBagItemConfig_basic,
				Check: testAccDataBagItemCheck(
					"chef_data_bag_item.test", &dataBagItemName,
				),
			},
		},
	})
}

func testAccDataBagItemCheck(rn string, name *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("data bag item id not set")
		}

		client := testAccProvider.Meta().(*chefc.Client)
		content, err := client.DataBags.GetItem("terraform-acc-test-bag-item-basic", rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting data bag item: %s", err)
		}

		expectedContent := map[string]interface{}{
			"id":             "terraform_acc_test",
			"something_else": true,
		}
		if !reflect.DeepEqual(content, expectedContent) {
			return fmt.Errorf("wrong content: expected %#v, got %#v", expectedContent, content)
		}

		if expected := "terraform_acc_test"; rs.Primary.Attributes["id"] != expected {
			return fmt.Errorf("wrong id; expected %#v, got %#v", expected, rs.Primary.Attributes["id"])
		}

		*name = rs.Primary.ID

		return nil
	}
}

func testAccDataBagItemCheckDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*chefc.Client)
		_, err := client.DataBags.GetItem("terraform-acc-test-bag-item-basic", name)
		if err == nil {
			return fmt.Errorf("data bag item still exists")
		}
		if _, ok := err.(*chefc.ErrorResponse); err != nil && !ok {
			return fmt.Errorf("got something other than an HTTP error (%v) when getting data bag item", err)
		}

		return nil
	}
}

const testAccDataBagItemConfig_basic = `
resource "chef_data_bag" "test" {
  name = "terraform-acc-test-bag-item-basic"
}
resource "chef_data_bag_item" "test" {
  data_bag_name = "terraform-acc-test-bag-item-basic"
  depends_on = ["chef_data_bag.test"]
  content_json = <<EOT
{
    "id": "terraform_acc_test",
    "something_else": true
}
EOT
}
`
