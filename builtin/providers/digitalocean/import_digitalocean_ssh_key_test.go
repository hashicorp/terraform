package digitalocean

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDigitalOceanSSHKey_importBasic(t *testing.T) {
	resourceName := "digitalocean_ssh_key.foobar"
	rInt := acctest.RandInt()
	publicKeyMaterial, _, err := acctest.RandSSHKeyPair("digitalocean@ssh-acceptance-test")
	if err != nil {
		t.Fatalf("Cannot generate test SSH key pair: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanSSHKeyConfig_basic(rInt, publicKeyMaterial),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
