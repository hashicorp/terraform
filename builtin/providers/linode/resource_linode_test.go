package linode

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/taoh/linodego"
)

func TestAccLinodeLinode_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLinodeLinodeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckLinodeLinodeConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLinodeLinodeExists("linode_linode.foobar"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "name", "foobar"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "size", "1024"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "image", "Ubuntu 14.04 LTS"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "region", "Dallas, TX, USA"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "kernel", "Latest 64 bit"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "group", "testing"),
				),
			},
		},
	})
}

func TestAccLinodeLinode_Update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLinodeLinodeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckLinodeLinodeConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLinodeLinodeExists("linode_linode.foobar"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "name", "foobar"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "group", "testing"),
				),
			},
			resource.TestStep{
				Config: testAccCheckLinodeLinodeConfig_updates,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLinodeLinodeExists("linode_linode.foobar"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "name", "foobaz"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "group", "integration"),
				),
			},
		},
	})
}

func TestAccLinodeLinode_PrivateNetworking(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLinodeLinodeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckLinodeLinodeConfig_PrivateNetworking,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLinodeLinodeExists("linode_linode.foobar"),
					testAccCheckLinodeLinodeAttributes_PrivateNetworking("linode_linode.foobar"),
					resource.TestCheckResourceAttr("linode_linode.foobar", "private_networking", "true"),
				),
			},
		},
	})
}

func testAccCheckLinodeLinodeDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*linodego.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_linode" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Failed to parse %s as int", rs.Primary.ID)
		}

		_, err = client.Linode.List(int(id))
		if err == nil {
			return fmt.Errorf("Found undelete linode %s", err)
		}
	}

	return nil
}

func testAccCheckLinodeLinodeExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", rs)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Linode id set")
		}

		client := testAccProvider.Meta().(*linodego.Client)
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			panic(err)
		}

		_, err = client.Linode.List(int(id))
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckLinodeLinodeAttributes_PrivateNetworking(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", rs)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Linode id set")
		}

		client := testAccProvider.Meta().(*linodego.Client)
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			panic(err)
		}
		_, err = client.Linode.List(int(id))
		if err != nil {
			return err
		}

		_, privateIp, err := getIps(client, int(id))
		if err != nil {
			return err
		}

		if privateIp == "" {
			return fmt.Errorf("Private Ip is not set")
		}
		return nil
	}
}

const testAccCheckLinodeLinodeConfig_basic = `
resource "linode_linode" "foobar" {
	name = "foobar"
	group = "testing"
	size = 1024
	image = "Ubuntu 14.04 LTS"
	region = "Dallas, TX, USA"
	kernel = "Latest 64 bit"
	root_password = "terraform-test"
	ssh_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCxtdizvJzTT38y2oXuoLUXbLUf9V0Jy9KsM0bgIvjUCSEbuLWCXKnWqgBmkv7iTKGZg3fx6JA10hiufdGHD7at5YaRUitGP2mvC2I68AYNZmLCGXh0hYMrrUB01OEXHaYhpSmXIBc9zUdTreL5CvYe3PAYzuBA0/lGFTnNsHosSd+suA4xfJWMr/Fr4/uxrpcy8N8BE16pm4kci5tcMh6rGUGtDEj6aE9k8OI4SRmSZJsNElsu/Z/K4zqCpkW/U06vOnRrE98j3NE07nxVOTqdAMZqopFiMP0MXWvd6XyS2/uKU+COLLc0+hVsgj+dVMTWfy8wZ58OJDsIKk/cI/7yF+GZz89Js+qYx7u9mNhpEgD4UrcRHpitlRgVhA8p6R4oBqb0m/rpKBd2BAFdcty3GIP9CWsARtsCbN6YDLJ1JN3xI34jSGC1ROktVHg27bEEiT5A75w3WJl96BlSo5zJsIZDTWlaqnr26YxNHba4ILdVLKigQtQpf8WFsnB9YzmDdb9K3w9szf5lAkb/SFXw+e+yPS9habkpOncL0oCsgag5wUGCEmZ7wpiY8QgARhuwsQUkxv1aUi/Nn7b7sAkKSkxtBI3LBXZ+vcUxZTH0ut4pe9rbrEed3ktAOF5FafjA1VtarPqqZ+g46xVO9llgpXcl3rVglFtXzTcUy09hGw== btobolaski@Brendans-MacBook-Pro.local"
}`

const testAccCheckLinodeLinodeConfig_updates = `
resource "linode_linode" "foobar" {
	name = "foobaz"
	group = "integration"
	size = 1024
	image = "Ubuntu 14.04 LTS"
	region = "Dallas, TX, USA"
	kernel = "Latest 64 bit"
	root_password = "terraform-test"
	ssh_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCxtdizvJzTT38y2oXuoLUXbLUf9V0Jy9KsM0bgIvjUCSEbuLWCXKnWqgBmkv7iTKGZg3fx6JA10hiufdGHD7at5YaRUitGP2mvC2I68AYNZmLCGXh0hYMrrUB01OEXHaYhpSmXIBc9zUdTreL5CvYe3PAYzuBA0/lGFTnNsHosSd+suA4xfJWMr/Fr4/uxrpcy8N8BE16pm4kci5tcMh6rGUGtDEj6aE9k8OI4SRmSZJsNElsu/Z/K4zqCpkW/U06vOnRrE98j3NE07nxVOTqdAMZqopFiMP0MXWvd6XyS2/uKU+COLLc0+hVsgj+dVMTWfy8wZ58OJDsIKk/cI/7yF+GZz89Js+qYx7u9mNhpEgD4UrcRHpitlRgVhA8p6R4oBqb0m/rpKBd2BAFdcty3GIP9CWsARtsCbN6YDLJ1JN3xI34jSGC1ROktVHg27bEEiT5A75w3WJl96BlSo5zJsIZDTWlaqnr26YxNHba4ILdVLKigQtQpf8WFsnB9YzmDdb9K3w9szf5lAkb/SFXw+e+yPS9habkpOncL0oCsgag5wUGCEmZ7wpiY8QgARhuwsQUkxv1aUi/Nn7b7sAkKSkxtBI3LBXZ+vcUxZTH0ut4pe9rbrEed3ktAOF5FafjA1VtarPqqZ+g46xVO9llgpXcl3rVglFtXzTcUy09hGw== btobolaski@Brendans-MacBook-Pro.local"
}`

const testAccCheckLinodeLinodeConfig_PrivateNetworking = `
resource "linode_linode" "foobar" {
	name = "foobaz"
	group = "integration"
	size = 1024
	image = "Ubuntu 14.04 LTS"
	region = "Dallas, TX, USA"
	kernel = "Latest 64 bit"
	root_password = "terraform-test"
	private_networking = true
	ssh_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCxtdizvJzTT38y2oXuoLUXbLUf9V0Jy9KsM0bgIvjUCSEbuLWCXKnWqgBmkv7iTKGZg3fx6JA10hiufdGHD7at5YaRUitGP2mvC2I68AYNZmLCGXh0hYMrrUB01OEXHaYhpSmXIBc9zUdTreL5CvYe3PAYzuBA0/lGFTnNsHosSd+suA4xfJWMr/Fr4/uxrpcy8N8BE16pm4kci5tcMh6rGUGtDEj6aE9k8OI4SRmSZJsNElsu/Z/K4zqCpkW/U06vOnRrE98j3NE07nxVOTqdAMZqopFiMP0MXWvd6XyS2/uKU+COLLc0+hVsgj+dVMTWfy8wZ58OJDsIKk/cI/7yF+GZz89Js+qYx7u9mNhpEgD4UrcRHpitlRgVhA8p6R4oBqb0m/rpKBd2BAFdcty3GIP9CWsARtsCbN6YDLJ1JN3xI34jSGC1ROktVHg27bEEiT5A75w3WJl96BlSo5zJsIZDTWlaqnr26YxNHba4ILdVLKigQtQpf8WFsnB9YzmDdb9K3w9szf5lAkb/SFXw+e+yPS9habkpOncL0oCsgag5wUGCEmZ7wpiY8QgARhuwsQUkxv1aUi/Nn7b7sAkKSkxtBI3LBXZ+vcUxZTH0ut4pe9rbrEed3ktAOF5FafjA1VtarPqqZ+g46xVO9llgpXcl3rVglFtXzTcUy09hGw== btobolaski@Brendans-MacBook-Pro.local"
}`
