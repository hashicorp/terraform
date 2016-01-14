package packet

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/packethost/packngo"
)

func TestAccPacketSSHKey_Basic(t *testing.T) {
	var key packngo.SSHKey

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPacketSSHKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPacketSSHKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPacketSSHKeyExists("packet_ssh_key.foobar", &key),
					testAccCheckPacketSSHKeyAttributes(&key),
					resource.TestCheckResourceAttr(
						"packet_ssh_key.foobar", "name", "foobar"),
					resource.TestCheckResourceAttr(
						"packet_ssh_key.foobar", "public_key", testAccValidPublicKey),
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

func testAccCheckPacketSSHKeyAttributes(key *packngo.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if key.Label != "foobar" {
			return fmt.Errorf("Bad name: %s", key.Label)
		}
		return nil
	}
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

var testAccCheckPacketSSHKeyConfig_basic = fmt.Sprintf(`
resource "packet_ssh_key" "foobar" {
    name = "foobar"
    public_key = "%s"
}`, testAccValidPublicKey)

var testAccValidPublicKey = strings.TrimSpace(`
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCKVmnMOlHKcZK8tpt3MP1lqOLAcqcJzhsvJcjscgVERRN7/9484SOBJ3HSKxxNG5JN8owAjy5f9yYwcUg+JaUVuytn5Pv3aeYROHGGg+5G346xaq3DAwX6Y5ykr2fvjObgncQBnuU5KHWCECO/4h8uWuwh/kfniXPVjFToc+gnkqA+3RKpAecZhFXwfalQ9mMuYGFxn+fwn8cYEApsJbsEmb0iJwPiZ5hjFC8wREuiTlhPHDgkBLOiycd20op2nXzDbHfCHInquEe/gYxEitALONxm0swBOwJZwlTDOB7C6y2dzlrtxr1L59m7pCkWI4EtTRLvleehBoj3u7jB4usR
`)
