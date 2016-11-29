package packet

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/packethost/packngo"
)

func TestAccPacketProject_Basic(t *testing.T) {
	var project packngo.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPacketProjectDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPacketProjectConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPacketProjectExists("packet_project.foobar", &project),
					testAccCheckPacketProjectAttributes(&project),
					resource.TestCheckResourceAttr(
						"packet_project.foobar", "name", "foobar"),
				),
			},
		},
	})
}

func testAccCheckPacketProjectDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*packngo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "packet_project" {
			continue
		}
		if _, _, err := client.Projects.Get(rs.Primary.ID); err == nil {
			return fmt.Errorf("Project still exists")
		}
	}

	return nil
}

func testAccCheckPacketProjectAttributes(project *packngo.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if project.Name != "foobar" {
			return fmt.Errorf("Bad name: %s", project.Name)
		}
		return nil
	}
}

func testAccCheckPacketProjectExists(n string, project *packngo.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*packngo.Client)

		foundProject, _, err := client.Projects.Get(rs.Primary.ID)
		if err != nil {
			return err
		}
		if foundProject.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found: %v - %v", rs.Primary.ID, foundProject)
		}

		*project = *foundProject

		return nil
	}
}

var testAccCheckPacketProjectConfig_basic = fmt.Sprintf(`
resource "packet_project" "foobar" {
    name = "foobar"
}`)
