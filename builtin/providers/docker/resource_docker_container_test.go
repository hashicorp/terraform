package docker

import (
	"fmt"
	"testing"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDockerContainer_basic(t *testing.T) {
	var c dc.Container
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
				),
			},
		},
	})
}

func TestAccDockerContainer_volume(t *testing.T) {
	var c dc.Container

	testCheck := func(*terraform.State) error {
		if len(c.Mounts) != 2 {
			return fmt.Errorf("Incorrect number of mounts: expected 2, got %d", len(c.Mounts))
		}

		for _, v := range c.Mounts {
			if v.Name != "testAccDockerContainerVolume_volume" {
				continue
			}

			if v.Destination != "/tmp/volume" {
				return fmt.Errorf("Bad destination on mount: expected /tmp/volume, got %q", v.Destination)
			}

			if v.Mode != "rw" {
				return fmt.Errorf("Bad mode on mount: expected rw, got %q", v.Mode)
			}

			return nil
		}

		return fmt.Errorf("Mount for testAccDockerContainerVolume_volume not found")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerVolumeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_customized(t *testing.T) {
	var c dc.Container

	testCheck := func(*terraform.State) error {
		if len(c.Config.Entrypoint) < 3 ||
			(c.Config.Entrypoint[0] != "/bin/bash" &&
				c.Config.Entrypoint[1] != "-c" &&
				c.Config.Entrypoint[2] != "ping localhost") {
			return fmt.Errorf("Container wrong entrypoint: %s", c.Config.Entrypoint)
		}

		if c.Config.User != "root:root" {
			return fmt.Errorf("Container wrong user: %s", c.Config.User)
		}

		if c.HostConfig.RestartPolicy.Name == "on-failure" {
			if c.HostConfig.RestartPolicy.MaximumRetryCount != 5 {
				return fmt.Errorf("Container has wrong restart policy max retry count: %d", c.HostConfig.RestartPolicy.MaximumRetryCount)
			}
		} else {
			return fmt.Errorf("Container has wrong restart policy: %s", c.HostConfig.RestartPolicy.Name)
		}

		if c.HostConfig.Memory != (512 * 1024 * 1024) {
			return fmt.Errorf("Container has wrong memory setting: %d", c.HostConfig.Memory)
		}

		if c.HostConfig.MemorySwap != (2048 * 1024 * 1024) {
			return fmt.Errorf("Container has wrong memory swap setting: %d", c.HostConfig.MemorySwap)
		}

		if c.HostConfig.CPUShares != 32 {
			return fmt.Errorf("Container has wrong cpu shares setting: %d", c.HostConfig.CPUShares)
		}

		if c.Config.Labels["env"] != "prod" || c.Config.Labels["role"] != "test" {
			return fmt.Errorf("Container does not have the correct labels")
		}

		if c.HostConfig.LogConfig.Type != "json-file" {
			return fmt.Errorf("Container does not have the correct log config: %s", c.HostConfig.LogConfig.Type)
		}

		if c.HostConfig.LogConfig.Config["max-size"] != "10m" {
			return fmt.Errorf("Container does not have the correct max-size log option: %v", c.HostConfig.LogConfig.Config["max-size"])
		}

		if c.HostConfig.LogConfig.Config["max-file"] != "20" {
			return fmt.Errorf("Container does not have the correct max-file log option: %v", c.HostConfig.LogConfig.Config["max-file"])
		}

		if len(c.HostConfig.ExtraHosts) != 2 {
			return fmt.Errorf("Container does not have correct number of extra host entries, got %d", len(c.HostConfig.ExtraHosts))
		}

		if c.HostConfig.ExtraHosts[0] != "testhost2:10.0.2.0" {
			return fmt.Errorf("Container has incorrect extra host string: %q", c.HostConfig.ExtraHosts[0])
		}

		if c.HostConfig.ExtraHosts[1] != "testhost:10.0.1.0" {
			return fmt.Errorf("Container has incorrect extra host string: %q", c.HostConfig.ExtraHosts[1])
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerCustomizedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func testAccContainerRunning(n string, container *dc.Container) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*dc.Client)
		containers, err := client.ListContainers(dc.ListContainersOptions{})
		if err != nil {
			return err
		}

		for _, c := range containers {
			if c.ID == rs.Primary.ID {
				inspected, err := client.InspectContainer(c.ID)
				if err != nil {
					return fmt.Errorf("Container could not be inspected: %s", err)
				}
				*container = *inspected
				return nil
			}
		}

		return fmt.Errorf("Container not found: %s", rs.Primary.ID)
	}
}

const testAccDockerContainerConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
}
`

const testAccDockerContainerVolumeConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_volume" "foo" {
    name = "testAccDockerContainerVolume_volume"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

    volumes {
        volume_name = "${docker_volume.foo.name}"
        container_path = "/tmp/volume"
        read_only = false
    }
}
`

const testAccDockerContainerCustomizedConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
	entrypoint = ["/bin/bash", "-c", "ping localhost"]
	user = "root:root"
	restart = "on-failure"
	max_retry_count = 5
	memory = 512
	memory_swap = 2048
	cpu_shares = 32
	labels {
		env = "prod"
		role = "test"
	}
	log_driver = "json-file"
	log_opts = {
		max-size = "10m"
		max-file = 20
	}
	network_mode = "bridge"

	host {
		host = "testhost"
		ip = "10.0.1.0"
	}

	host {
		host = "testhost2"
		ip = "10.0.2.0"
	}
}
`
