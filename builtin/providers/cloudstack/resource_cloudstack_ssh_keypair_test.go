package cloudstack

import (
	"fmt"
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
					testAccCheckCloudStackSSHKeyPairCreateAttributes("terraform-test-keypair"),
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

		if rs.Primary.ID == "" {
			return fmt.Errorf("No key pair ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		p := cs.SSH.NewListSSHKeyPairsParams()
		p.SetName(rs.Primary.ID)

		list, err := cs.SSH.ListSSHKeyPairs(p)
		if err != nil {
			return err
		}

		if list.Count != 1 || list.SSHKeyPairs[0].Name != rs.Primary.ID {
			return fmt.Errorf("Key pair not found")
		}

		*sshkey = *list.SSHKeyPairs[0]

		return nil
	}
}

func testAccCheckCloudStackSSHKeyPairAttributes(
	keypair *cloudstack.SSHKeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		fpLen := len(keypair.Fingerprint)
		if fpLen != 47 {
			return fmt.Errorf("SSH key: Attribute private_key expected length 47, got %d", fpLen)
		}

		return nil
	}
}

func testAccCheckCloudStackSSHKeyPairCreateAttributes(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		found := false

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "cloudstack_ssh_keypair" {
				continue
			}

			if rs.Primary.ID != name {
				continue
			}

			if !strings.Contains(rs.Primary.Attributes["private_key"], "PRIVATE KEY") {
				return fmt.Errorf(
					"SSH key: Attribute private_key expected 'PRIVATE KEY' to be present, got %s",
					rs.Primary.Attributes["private_key"])
			}

			found = true
			break
		}

		if !found {
			return fmt.Errorf("Could not find key pair %s", name)
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

		if rs.Primary.ID == "" {
			return fmt.Errorf("No key pair ID is set")
		}

		p := cs.SSH.NewListSSHKeyPairsParams()
		p.SetName(rs.Primary.ID)

		list, err := cs.SSH.ListSSHKeyPairs(p)
		if err != nil {
			return err
		}

		for _, keyPair := range list.SSHKeyPairs {
			if keyPair.Name == rs.Primary.ID {
				return fmt.Errorf("Key pair %s still exists", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccCloudStackSSHKeyPair_create = fmt.Sprintf(`
resource "cloudstack_ssh_keypair" "foo" {
  name = "terraform-test-keypair"
}`)

var testAccCloudStackSSHKeyPair_register = fmt.Sprintf(`
resource "cloudstack_ssh_keypair" "foo" {
  name = "terraform-test-keypair"
  public_key = "%s"
}`, CLOUDSTACK_SSH_PUBLIC_KEY)
