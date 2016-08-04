package digitalocean

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDigitalOceanSSHKey_importBasic(t *testing.T) {
	resourceName := "digitalocean_ssh_key.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanSSHKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanSSHKeyConfig_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
