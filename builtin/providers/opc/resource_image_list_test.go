package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCImageList_Basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccImageList_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckImageListDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckImageListExists,
			},
		},
	})
}

func TestAccOPCImageList_Complete(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccImageList_complete, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckImageListDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckImageListExists,
			},
		},
	})
}

func testAccCheckImageListExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).ImageList()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_image_list" {
			continue
		}

		input := compute.GetImageListInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetImageList(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Image List %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckImageListDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).ImageList()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_image_list" {
			continue
		}

		input := compute.GetImageListInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetImageList(&input); err == nil {
			return fmt.Errorf("Image List %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

var testAccImageList_basic = `
resource "opc_compute_image_list" "test" {
  name        = "test-acc-image-list-basic-%d"
  description = "Image List (Basic)"
}
`

var testAccImageList_complete = `
resource "opc_compute_image_list" "test" {
  name        = "test-acc-image-list-complete-%d"
  description = "Image List (Complete)"
  default     = 2
}
`
