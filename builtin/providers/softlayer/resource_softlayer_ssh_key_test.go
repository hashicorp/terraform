package softlayer

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

func TestAccSoftLayerSSHKey_Basic(t *testing.T) {
	var key datatypes.SoftLayer_Security_Ssh_Key

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSoftLayerSSHKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerSSHKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerSSHKeyExists("softlayer_ssh_key.testacc_foobar", &key),
					testAccCheckSoftLayerSSHKeyAttributes(&key),
					resource.TestCheckResourceAttr(
						"softlayer_ssh_key.testacc_foobar", "name", "testacc_foobar"),
					resource.TestCheckResourceAttr(
						"softlayer_ssh_key.testacc_foobar", "public_key", testAccValidPublicKey),
					resource.TestCheckResourceAttr(
						"softlayer_ssh_key.testacc_foobar", "notes", "first_note"),
				),
			},

			resource.TestStep{
				Config: testAccCheckSoftLayerSSHKeyConfig_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerSSHKeyExists("softlayer_ssh_key.testacc_foobar", &key),
					resource.TestCheckResourceAttr(
						"softlayer_ssh_key.testacc_foobar", "name", "changed_name"),
					resource.TestCheckResourceAttr(
						"softlayer_ssh_key.testacc_foobar", "public_key", testAccValidPublicKey),
					resource.TestCheckResourceAttr(
						"softlayer_ssh_key.testacc_foobar", "notes", "changed_note"),
				),
			},
		},
	})
}

func testAccCheckSoftLayerSSHKeyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).sshKeyService

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "softlayer_ssh_key" {
			continue
		}

		keyId, _ := strconv.Atoi(rs.Primary.ID)

		// Try to find the key
		_, err := client.GetObject(keyId)

		if err == nil {
			return fmt.Errorf("SSH key still exists")
		}
	}

	return nil
}

func testAccCheckSoftLayerSSHKeyAttributes(key *datatypes.SoftLayer_Security_Ssh_Key) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if key.Label != "testacc_foobar" {
			return fmt.Errorf("Bad name: %s", key.Label)
		}

		return nil
	}
}

func testAccCheckSoftLayerSSHKeyExists(n string, key *datatypes.SoftLayer_Security_Ssh_Key) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		keyId, _ := strconv.Atoi(rs.Primary.ID)

		client := testAccProvider.Meta().(*Client).sshKeyService
		foundKey, err := client.GetObject(keyId)

		if err != nil {
			return err
		}

		if strconv.Itoa(int(foundKey.Id)) != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*key = foundKey

		return nil
	}
}

var testAccCheckSoftLayerSSHKeyConfig_basic = fmt.Sprintf(`
resource "softlayer_ssh_key" "testacc_foobar" {
    name = "testacc_foobar"
    notes = "first_note"
    public_key = "%s"
}`, testAccValidPublicKey)

var testAccCheckSoftLayerSSHKeyConfig_updated = fmt.Sprintf(`
resource "softlayer_ssh_key" "testacc_foobar" {
    name = "changed_name"
    notes = "changed_note"
    public_key = "%s"
}`, testAccValidPublicKey)

var testAccValidPublicKey = strings.TrimSpace(`
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCKVmnMOlHKcZK8tpt3MP1lqOLAcqcJzhsvJcjscgVERRN7/9484SOBJ3HSKxxNG5JN8owAjy5f9yYwcUg+JaUVuytn5Pv3aeYROHGGg+5G346xaq3DAwX6Y5ykr2fvjObgncQBnuU5KHWCECO/4h8uWuwh/kfniXPVjFToc+gnkqA+3RKpAecZhFXwfalQ9mMuYGFxn+fwn8cYEApsJbsEmb0iJwPiZ5hjFC8wREuiTlhPHDgkBLOiycd20op2nXzDbHfCHInquEe/gYxEitALONxm0swBOwJZwlTDOB7C6y2dzlrtxr1L59m7pCkWI4EtTRLvleehBoj3u7jB4usR
`)
