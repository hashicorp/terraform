package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanSSHKey_Basic(t *testing.T) {
	var key godo.Key
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
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanSSHKeyExists("digitalocean_ssh_key.foobar", &key),
					resource.TestCheckResourceAttr(
						"digitalocean_ssh_key.foobar", "name", fmt.Sprintf("foobar-%d", rInt)),
					resource.TestCheckResourceAttr(
						"digitalocean_ssh_key.foobar", "public_key", publicKeyMaterial),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanSSHKeyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_ssh_key" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		// Try to find the key
		_, _, err = client.Keys.GetByID(context.Background(), id)

		if err == nil {
			return fmt.Errorf("SSH key still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanSSHKeyExists(n string, key *godo.Key) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		// Try to find the key
		foundKey, _, err := client.Keys.GetByID(context.Background(), id)

		if err != nil {
			return err
		}

		if strconv.Itoa(foundKey.ID) != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*key = *foundKey

		return nil
	}
}

func testAccCheckDigitalOceanSSHKeyConfig_basic(rInt int, key string) string {
	return fmt.Sprintf(`
resource "digitalocean_ssh_key" "foobar" {
    name = "foobar-%d"
    public_key = "%s"
}`, rInt, key)
}
