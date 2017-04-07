package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCImageListEntry_Basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccImageListEntry_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckImageListEntryDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckImageListEntryExists,
			},
		},
	})
}

func TestAccOPCImageListEntry_Complete(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccImageListEntry_Complete, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckImageListEntryDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckImageListEntryExists,
			},
		},
	})
}

func TestAccOPCImageListEntry_CompleteExpanded(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccImageListEntry_CompleteExpanded, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckImageListEntryDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckImageListEntryExists,
			},
		},
	})
}

func testAccCheckImageListEntryExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).ImageListEntries()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_image_list_entry" {
			continue
		}

		name, version, err := parseOPCImageListEntryID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing the Image List ID: '%s': %+v", rs.Primary.ID, err)
		}

		input := compute.GetImageListEntryInput{
			Name:    *name,
			Version: *version,
		}

		if _, err := client.GetImageListEntry(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Image List Entry %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckImageListEntryDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).ImageListEntries()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_image_list_entry" {
			continue
		}

		name, version, err := parseOPCImageListEntryID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing the Image List ID: %+v", err)
		}

		input := compute.GetImageListEntryInput{
			Name:    *name,
			Version: *version,
		}
		if info, err := client.GetImageListEntry(&input); err == nil {
			return fmt.Errorf("Image List Entry %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

var testAccImageListEntry_basic = `
resource "opc_compute_image_list" "test" {
  name        = "test-acc-image-list-entry-basic-%d"
  description = "Acceptance Test TestAccOPCImageListEntry_Basic"
  default     = 1
}

resource "opc_compute_image_list_entry" "test" {
  name           = "${opc_compute_image_list.test.name}"
  machine_images = [ "/oracle/public/oel_6.7_apaas_16.4.5_1610211300" ]
  version        = 1
}
`

var testAccImageListEntry_Complete = `
resource "opc_compute_image_list" "test" {
  name        = "test-acc-image-list-entry-basic-%d"
  description = "Acceptance Test TestAccOPCImageListEntry_Basic"
  default     = 1
}

resource "opc_compute_image_list_entry" "test" {
  name           = "${opc_compute_image_list.test.name}"
  machine_images = [ "/oracle/public/oel_6.7_apaas_16.4.5_1610211300" ]
  attributes     = "{\"hello\":\"world\"}"
  version        = 1
}
`

var testAccImageListEntry_CompleteExpanded = `
resource "opc_compute_image_list" "test" {
  name        = "test-acc-image-list-entry-basic-%d"
  description = "Acceptance Test TestAccOPCImageListEntry_Basic"
  default     = 1
}

resource "opc_compute_image_list_entry" "test" {
  name           = "${opc_compute_image_list.test.name}"
  machine_images = [ "/oracle/public/oel_6.7_apaas_16.4.5_1610211300" ]
  attributes     = <<JSON
  {
    "hello": "world"
  }
JSON
  version        = 1
}
`
