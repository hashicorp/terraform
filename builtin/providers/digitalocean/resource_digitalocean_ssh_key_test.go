package digitalocean

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanSSHKey_Basic(t *testing.T) {
	var key godo.Key

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanSSHKeyConfig_basic(testAccValidPublicKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanSSHKeyExists("digitalocean_ssh_key.foobar", &key),
					testAccCheckDigitalOceanSSHKeyAttributes(&key),
					resource.TestCheckResourceAttr(
						"digitalocean_ssh_key.foobar", "name", "foobar"),
					resource.TestCheckResourceAttr(
						"digitalocean_ssh_key.foobar", "public_key", strings.TrimSpace(testAccValidPublicKey)),
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
		_, _, err = client.Keys.GetByID(id)

		if err == nil {
			return fmt.Errorf("SSH key still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanSSHKeyAttributes(key *godo.Key) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if key.Name != "foobar" {
			return fmt.Errorf("Bad name: %s", key.Name)
		}

		return nil
	}
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
		foundKey, _, err := client.Keys.GetByID(id)

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

func testAccCheckDigitalOceanSSHKeyConfig_basic(key string) string {
	return fmt.Sprintf(`
resource "digitalocean_ssh_key" "foobar" {
    name = "foobar"
    public_key = "%s"
}`, key)
}

var testAccValidPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCKVmnMOlHKcZK8tpt3MP1lqOLAcqcJzhsvJcjscgVERRN7/9484SOBJ3HSKxxNG5JN8owAjy5f9yYwcUg+JaUVuytn5Pv3aeYROHGGg+5G346xaq3DAwX6Y5ykr2fvjObgncQBnuU5KHWCECO/4h8uWuwh/kfniXPVjFToc+gnkqA+3RKpAecZhFXwfalQ9mMuYGFxn+fwn8cYEApsJbsEmb0iJwPiZ5hjFC8wREuiTlhPHDgkBLOiycd20op2nXzDbHfCHInquEe/gYxEitALONxm0swBOwJZwlTDOB7C6y2dzlrtxr1L59m7pCkWI4EtTRLvleehBoj3u7jB4usR`
var testAccValidImportPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCwelf/LV8TKOd6ZCcDwU9L8YRdVwfR2q8E+Bzamcxwb1U41vnfyvEZbzx0aeXimdHipOql0SG2tu9Z+bzekROVc13OP/gtGRlWwZ9RoKE8hFHanhi0K2tC6OWagsvmHpW/xptsYAo2k+eRJJo0iy/hLNG2c1v5rrjg6xwnSL3+a7bFM4xNDux5sNYCmxIBfIL+4rQ8XBlxsjMrGoev/uumZ0yc75JtBCOSZbdie936pvVmoAf4nhxNbe5lOxp+18zHhBbO2fjhux4xmf4hLM2gHsdBGqtnphzLh3d1+uMIpv7ZMTKN7pBw53xQxw7hhDYuNKc8FkQ8xK6IL5bu/Ar/`
