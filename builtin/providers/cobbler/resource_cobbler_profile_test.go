package cobbler

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	cobbler "github.com/jtopjian/cobblerclient"
)

func TestAccCobblerProfile_basic(t *testing.T) {
	var distro cobbler.Distro
	var profile cobbler.Profile

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCobblerPreCheck(t) },
		Providers:    testAccCobblerProviders,
		CheckDestroy: testAccCobblerCheckProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCobblerProfile_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCobblerCheckDistroExists(t, "cobbler_distro.foo", &distro),
					testAccCobblerCheckProfileExists(t, "cobbler_profile.foo", &profile),
				),
			},
		},
	})
}

func TestAccCobblerProfile_change(t *testing.T) {
	var distro cobbler.Distro
	var profile cobbler.Profile

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCobblerPreCheck(t) },
		Providers:    testAccCobblerProviders,
		CheckDestroy: testAccCobblerCheckProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCobblerProfile_change_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCobblerCheckDistroExists(t, "cobbler_distro.foo", &distro),
					testAccCobblerCheckProfileExists(t, "cobbler_profile.foo", &profile),
				),
			},
			resource.TestStep{
				Config: testAccCobblerProfile_change_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCobblerCheckDistroExists(t, "cobbler_distro.foo", &distro),
					testAccCobblerCheckProfileExists(t, "cobbler_profile.foo", &profile),
				),
			},
		},
	})
}

func TestAccCobblerProfile_withRepo(t *testing.T) {
	var distro cobbler.Distro
	var profile cobbler.Profile

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCobblerPreCheck(t) },
		Providers:    testAccCobblerProviders,
		CheckDestroy: testAccCobblerCheckProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCobblerProfile_withRepo,
				Check: resource.ComposeTestCheckFunc(
					testAccCobblerCheckDistroExists(t, "cobbler_distro.foo", &distro),
					testAccCobblerCheckProfileExists(t, "cobbler_profile.foo", &profile),
				),
			},
		},
	})
}

func testAccCobblerCheckProfileDestroy(s *terraform.State) error {
	config := testAccCobblerProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cobbler_profile" {
			continue
		}

		if _, err := config.cobblerClient.GetProfile(rs.Primary.ID); err == nil {
			return fmt.Errorf("Profile still exists")
		}
	}

	return nil
}

func testAccCobblerCheckProfileExists(t *testing.T, n string, profile *cobbler.Profile) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccCobblerProvider.Meta().(*Config)

		found, err := config.cobblerClient.GetProfile(rs.Primary.ID)
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Profile not found")
		}

		*profile = *found

		return nil
	}
}

var testAccCobblerProfile_basic = `
	resource "cobbler_distro" "foo" {
		name = "foo"
		breed = "ubuntu"
		os_version = "trusty"
		arch = "x86_64"
		kernel = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/linux"
		initrd = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/initrd.gz"
	}

	resource "cobbler_profile" "foo" {
		name = "foo"
		distro = "${cobbler_distro.foo.name}"
	}`

var testAccCobblerProfile_change_1 = `
	resource "cobbler_distro" "foo" {
		name = "foo"
		breed = "ubuntu"
		os_version = "trusty"
		arch = "x86_64"
		kernel = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/linux"
		initrd = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/initrd.gz"
	}

	resource "cobbler_profile" "foo" {
		name = "foo"
		comment = "I am a profile"
		distro = "${cobbler_distro.foo.name}"
	}`

var testAccCobblerProfile_change_2 = `
	resource "cobbler_distro" "foo" {
		name = "foo"
		breed = "ubuntu"
		os_version = "trusty"
		arch = "x86_64"
		kernel = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/linux"
		initrd = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/initrd.gz"
	}

	resource "cobbler_profile" "foo" {
		name = "foo"
		comment = "I am a profile again"
		distro = "${cobbler_distro.foo.name}"
	}`

var testAccCobblerProfile_withRepo = `
	resource "cobbler_distro" "foo" {
		name = "foo"
		breed = "ubuntu"
		os_version = "trusty"
		arch = "x86_64"
		kernel = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/linux"
		initrd = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/initrd.gz"
	}

	resource "cobbler_profile" "foo" {
		name = "foo"
		comment = "I am a profile again"
		distro = "${cobbler_distro.foo.name}"
		repos = ["Ubuntu-14.04-x86_64"]
	}`
