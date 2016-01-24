package docker

import (
	"fmt"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerVolume_basic(t *testing.T) {
	var v dc.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerVolumeConfig,
				Check: resource.ComposeTestCheckFunc(
					checkDockerVolume("docker_volume.foo", &v),
					resource.TestCheckResourceAttr("docker_volume.foo", "id", "testAccDockerVolume_basic"),
					resource.TestCheckResourceAttr("docker_volume.foo", "name", "testAccDockerVolume_basic"),
				),
			},
		},
	})
}

func checkDockerVolume(n string, volume *dc.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*dc.Client)
		volumes, err := client.ListVolumes(dc.ListVolumesOptions{})
		if err != nil {
			return err
		}

		for _, v := range volumes {
			if v.Name == rs.Primary.ID {
				inspected, err := client.InspectVolume(v.Name)
				if err != nil {
					return fmt.Errorf("Volume could not be inspected: %s", err)
				}
				*volume = *inspected
				return nil
			}
		}

		return fmt.Errorf("Volume not found: %s", rs.Primary.ID)
	}
}

const testAccDockerVolumeConfig = `
resource "docker_volume" "foo" {
	name = "testAccDockerVolume_basic"
}
`
