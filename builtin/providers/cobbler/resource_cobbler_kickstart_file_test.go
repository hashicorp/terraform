package cobbler

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	cobbler "github.com/jtopjian/cobblerclient"
)

func TestAccCobblerKickstartFile_basic(t *testing.T) {
	var ks cobbler.KickstartFile

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCobblerPreCheck(t) },
		Providers:    testAccCobblerProviders,
		CheckDestroy: testAccCobblerCheckKickstartFileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCobblerKickstartFile_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCobblerCheckKickstartFileExists(t, "cobbler_kickstart_file.foo", &ks),
				),
			},
		},
	})
}

func testAccCobblerCheckKickstartFileDestroy(s *terraform.State) error {
	config := testAccCobblerProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cobbler_kickstart_file" {
			continue
		}

		if _, err := config.cobblerClient.GetKickstartFile(rs.Primary.ID); err == nil {
			return fmt.Errorf("Kickstart File still exists")
		}
	}

	return nil
}

func testAccCobblerCheckKickstartFileExists(t *testing.T, n string, ks *cobbler.KickstartFile) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccCobblerProvider.Meta().(*Config)

		found, err := config.cobblerClient.GetKickstartFile(rs.Primary.ID)
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Kickstart File not found")
		}

		*ks = *found

		return nil
	}
}

var testAccCobblerKickstartFile_basic = `
	resource "cobbler_kickstart_file" "foo" {
		name = "/var/lib/cobbler/kickstarts/foo.ks"
		body = "I'm a kickstart file."
	}`
