package digitalocean

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDigitalOceanTag_importBasic(t *testing.T) {
	resourceName := "digitalocean_tag.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanTagDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanTagConfig_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
