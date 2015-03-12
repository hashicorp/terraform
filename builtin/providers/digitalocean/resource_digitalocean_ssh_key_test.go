package digitalocean

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/digitalocean"
)

func TestAccDigitalOceanSSHKey_Basic(t *testing.T) {
	var key digitalocean.SSHKey

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanSSHKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanSSHKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanSSHKeyExists("digitalocean_ssh_key.foobar", &key),
					testAccCheckDigitalOceanSSHKeyAttributes(&key),
					resource.TestCheckResourceAttr(
						"digitalocean_ssh_key.foobar", "name", "foobar"),
					resource.TestCheckResourceAttr(
						"digitalocean_ssh_key.foobar", "public_key", "abcdef"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanSSHKeyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*digitalocean.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_ssh_key" {
			continue
		}

		// Try to find the key
		_, err := client.RetrieveSSHKey(rs.Primary.ID)

		if err == nil {
			fmt.Errorf("SSH key still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanSSHKeyAttributes(key *digitalocean.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if key.Name != "foobar" {
			return fmt.Errorf("Bad name: %s", key.Name)
		}

		return nil
	}
}

func testAccCheckDigitalOceanSSHKeyExists(n string, key *digitalocean.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*digitalocean.Client)

		foundKey, err := client.RetrieveSSHKey(rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundKey.Name != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*key = foundKey

		return nil
	}
}

const testAccCheckDigitalOceanSSHKeyConfig_basic = `
resource "digitalocean_ssh_key" "foobar" {
    name = "foobar"
    public_key = "abcdef"
}`
