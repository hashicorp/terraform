package cloudstack

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackSSHKeyPair_basic(t *testing.T) {
	var sshkey cloudstack.SSHKeyPair

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackSSHKeyPairDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackSSHKeyPair_create,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackSSHKeyPairExists("cloudstack_ssh_keypair.foo", &sshkey),
					testAccCheckCloudStackSSHKeyPairAttributes(&sshkey),
					testAccCheckCloudStackSSHKeyPairCreateAttributes("cloudstack_ssh_keypair.foo"),
				),
			},
		},
	})
}

func TestAccCloudStackSSHKeyPair_register(t *testing.T) {
	var sshkey cloudstack.SSHKeyPair

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackSSHKeyPairDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackSSHKeyPair_register,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackSSHKeyPairExists("cloudstack_ssh_keypair.foo", &sshkey),
					testAccCheckCloudStackSSHKeyPairAttributes(&sshkey),
					resource.TestCheckResourceAttr(
						"cloudstack_ssh_keypair.foo",
						"public_key",
						CLOUDSTACK_SSH_PUBLIC_KEY),
				),
			},
		},
	})
}

func testAccCheckCloudStackSSHKeyPairExists(n string, sshkey *cloudstack.SSHKeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.Attributes["name"] == "" {
			return fmt.Errorf("No ssh key name is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		p := cs.SSH.NewListSSHKeyPairsParams()
		p.SetName(rs.Primary.Attributes["name"])
		list, err := cs.SSH.ListSSHKeyPairs(p)

		if err != nil {
			return err
		}

		if list.Count == 1 && list.SSHKeyPairs[0].Name == rs.Primary.Attributes["name"] {
			//ssh key exists
			*sshkey = *list.SSHKeyPairs[0]
			return nil
		}

		return fmt.Errorf("SSH key not found")
	}
}

func testAccCheckCloudStackSSHKeyPairAttributes(
	sshkey *cloudstack.SSHKeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		fingerprintLen := len(sshkey.Fingerprint)
		if fingerprintLen != 47 {
			return fmt.Errorf(
				"SSH key: Attribute private_key expected length 47, got %d",
				fingerprintLen)
		}

		return nil
	}
}

func testAccCheckCloudStackSSHKeyPairCreateAttributes(
	name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		log.Printf("Private key calculated: %s", is.Attributes["private_key"])
		if !strings.Contains(is.Attributes["private_key"], "PRIVATE KEY") {
			return fmt.Errorf(
				"SSH key: Attribute private_key expected 'PRIVATE KEY' to be present, got %s",
				is.Attributes["private_key"])
		}
		return nil
	}
}

func testAccCheckCloudStackSSHKeyPairDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_ssh_keypair" {
			continue
		}

		if rs.Primary.Attributes["name"] == "" {
			return fmt.Errorf("No ssh key name is set")
		}

		p := cs.SSH.NewDeleteSSHKeyPairParams(rs.Primary.Attributes["name"])
		_, err := cs.SSH.DeleteSSHKeyPair(p)

		if err != nil {
			return fmt.Errorf(
				"Error deleting ssh key (%s): %s",
				rs.Primary.Attributes["name"], err)
		}
	}

	return nil
}

var testAccCloudStackSSHKeyPair_create = fmt.Sprintf(`
resource "cloudstack_ssh_keypair" "foo" {
  name = "terraform-testacc"
}`)

var testAccCloudStackSSHKeyPair_register = fmt.Sprintf(`
resource "cloudstack_ssh_keypair" "foo" {
  name = "terraform-testacc"
  public_key = "%s"
}`, CLOUDSTACK_SSH_PUBLIC_KEY)
