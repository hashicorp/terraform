package packet

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/packethost/packngo"
)

func TestAccPacketSSHKey_Basic(t *testing.T) {
	var key packngo.SSHKey
	rInt := acctest.RandInt()
	publicKeyMaterial, _, err := acctest.RandSSHKeyPair("")
	if err != nil {
		t.Fatalf("Cannot generate test SSH key pair: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPacketSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPacketSSHKeyConfig_basic(rInt, publicKeyMaterial),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPacketSSHKeyExists("packet_ssh_key.foobar", &key),
					resource.TestCheckResourceAttr(
						"packet_ssh_key.foobar", "name", fmt.Sprintf("foobar-%d", rInt)),
					resource.TestCheckResourceAttr(
						"packet_ssh_key.foobar", "public_key", publicKeyMaterial),
				),
			},
		},
	})
}

func testAccCheckPacketSSHKeyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*packngo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "packet_ssh_key" {
			continue
		}
		if _, _, err := client.SSHKeys.Get(rs.Primary.ID); err == nil {
			return fmt.Errorf("SSH key still exists")
		}
	}

	return nil
}

func testAccCheckPacketSSHKeyExists(n string, key *packngo.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*packngo.Client)

		foundKey, _, err := client.SSHKeys.Get(rs.Primary.ID)
		if err != nil {
			return err
		}
		if foundKey.ID != rs.Primary.ID {
			return fmt.Errorf("SSh Key not found: %v - %v", rs.Primary.ID, foundKey)
		}

		*key = *foundKey

		fmt.Printf("key: %v", key)
		return nil
	}
}

func testAccCheckPacketSSHKeyConfig_basic(rInt int, publicSshKey string) string {
	return fmt.Sprintf(`
resource "packet_ssh_key" "foobar" {
    name = "foobar-%d"
    public_key = "%s"
}`, rInt, publicSshKey)
}
