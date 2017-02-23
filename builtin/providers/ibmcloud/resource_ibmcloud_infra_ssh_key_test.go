package ibmcloud

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/services"

	"github.com/hashicorp/terraform/helper/acctest"
)

func TestAccIBMCloudInfraSSHKey_basic(t *testing.T) {
	var key datatypes.Security_Ssh_Key

	label1 := fmt.Sprintf("ssh_key_test_create_step_label_%d", acctest.RandInt())
	label2 := fmt.Sprintf("ssh_key_test_update_step_label_%d", acctest.RandInt())
	notes1 := fmt.Sprintf("ssh_key_test_create_step_notes_%d", acctest.RandInt())
	notes2 := fmt.Sprintf("ssh_key_test_update_step_notes_%d", acctest.RandInt())

	publicKey := strings.TrimSpace(`
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCKVmnMOlHKcZK8tpt3MP1lqOLAcqcJzhsvJcjscgVERRN7/9484SOBJ3HSKxxNG5JN8owAjy5f9yYwcUg+JaUVuytn5Pv3aeYROHGGg+5G346xaq3DAwX6Y5ykr2fvjObgncQBnuU5KHWCECO/4h8uWuwh/kfniXPVjFToc+gnkqA+3RKpAecZhFXwfalQ9mMuYGFxn+fwn8cYEApsJbsEmb0iJwPiZ5hjFC8wREuiTlhPHDgkBLOiycd20op2nXzDbHfCHInquEe/gYxEitALONxm0swBOwJZwlTDOB7C6y2dzlrtxr1L59m7pCkWI4EtTRLvleehBoj3u7jB4usR
`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIBMCloudInfraSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIBMCloudInfraSSHKeyConfig(label1, notes1, publicKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIBMCloudInfraSSHKeyExists("ibmcloud_infra_ssh_key.testacc_ssh_key", &key),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_ssh_key.testacc_ssh_key", "label", label1),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_ssh_key.testacc_ssh_key", "public_key", publicKey),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_ssh_key.testacc_ssh_key", "notes", notes1),
				),
			},

			{
				Config: testAccCheckIBMCloudInfraSSHKeyConfig(label2, notes2, publicKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIBMCloudInfraSSHKeyExists("ibmcloud_infra_ssh_key.testacc_ssh_key", &key),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_ssh_key.testacc_ssh_key", "label", label2),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_ssh_key.testacc_ssh_key", "public_key", publicKey),
					resource.TestCheckResourceAttr(
						"ibmcloud_infra_ssh_key.testacc_ssh_key", "notes", notes2),
				),
			},
		},
	})
}

func testAccCheckIBMCloudInfraSSHKeyDestroy(s *terraform.State) error {
	service := services.GetSecuritySshKeyService(testAccProvider.Meta().(ClientSession).SoftLayerSession())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ibmcloud_infra_ssh_key" {
			continue
		}

		keyID, _ := strconv.Atoi(rs.Primary.ID)

		// Try to find the key
		_, err := service.Id(keyID).GetObject()

		if err == nil {
			return fmt.Errorf("SSH key %d still exists", keyID)
		}
	}

	return nil
}

func testAccCheckIBMCloudInfraSSHKeyExists(n string, key *datatypes.Security_Ssh_Key) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Record ID is set")
		}

		keyID, _ := strconv.Atoi(rs.Primary.ID)

		service := services.GetSecuritySshKeyService(testAccProvider.Meta().(ClientSession).SoftLayerSession())
		foundKey, err := service.Id(keyID).GetObject()

		if err != nil {
			return err
		}

		if strconv.Itoa(int(*foundKey.Id)) != rs.Primary.ID {
			return fmt.Errorf("Record %d not found", keyID)
		}

		*key = foundKey

		return nil
	}
}

func testAccCheckIBMCloudInfraSSHKeyConfig(label, notes, publicKey string) string {
	return fmt.Sprintf(`
resource "ibmcloud_infra_ssh_key" "testacc_ssh_key" {
    label = "%s"
    notes = "%s"
    public_key = "%s"
}`, label, notes, publicKey)

}
