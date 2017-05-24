package docker

import (
	"archive/tar"
	"bytes"
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

func TestAccDockerContainerPath_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "/var/log", ErrCount: 0},
		{Value: "/tmp", ErrCount: 0},
		{Value: "C:\\Windows\\System32", ErrCount: 0},
		{Value: "C:\\Program Files\\MSBuild", ErrCount: 0},
		{Value: "test", ErrCount: 1},
		{Value: "C:Test", ErrCount: 1},
		{Value: "", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateDockerContainerPath(tc.Value, "docker_container")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Docker Container Path to trigger a validation error")
		}
	}
}

func TestAccDockerContainer_volume(t *testing.T) {
	var c dc.Container

	testCheck := func(*terraform.State) error {
		if len(c.Mounts) != 1 {
			return fmt.Errorf("Incorrect number of mounts: expected 1, got %d", len(c.Mounts))
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
			return fmt.Errorf("Container has wrong memory swap setting: %d\n\r\tPlease check that you machine supports memory swap (you can do that by running 'docker info' command).", c.HostConfig.MemorySwap)
		}

		if c.HostConfig.CPUShares != 32 {
			return fmt.Errorf("Container has wrong cpu shares setting: %d", c.HostConfig.CPUShares)
		}

		if len(c.HostConfig.DNS) != 1 {
			return fmt.Errorf("Container does not have the correct number of dns entries: %d", len(c.HostConfig.DNS))
		}

		if c.HostConfig.DNS[0] != "8.8.8.8" {
			return fmt.Errorf("Container has wrong dns setting: %v", c.HostConfig.DNS[0])
		}

		if len(c.HostConfig.DNSOptions) != 1 {
			return fmt.Errorf("Container does not have the correct number of dns option entries: %d", len(c.HostConfig.DNS))
		}

		if c.HostConfig.DNSOptions[0] != "rotate" {
			return fmt.Errorf("Container has wrong dns option setting: %v", c.HostConfig.DNS[0])
		}

		if len(c.HostConfig.DNSSearch) != 1 {
			return fmt.Errorf("Container does not have the correct number of dns search entries: %d", len(c.HostConfig.DNS))
		}

		if c.HostConfig.DNSSearch[0] != "example.com" {
			return fmt.Errorf("Container has wrong dns search setting: %v", c.HostConfig.DNS[0])
		}

		if len(c.HostConfig.CapAdd) != 1 {
			return fmt.Errorf("Container does not have the correct number of Capabilities in ADD: %d", len(c.HostConfig.CapAdd))
		}

		if c.HostConfig.CapAdd[0] != "ALL" {
			return fmt.Errorf("Container has wrong CapAdd setting: %v", c.HostConfig.CapAdd[0])
		}

		if len(c.HostConfig.CapDrop) != 1 {
			return fmt.Errorf("Container does not have the correct number of Capabilities in Drop: %d", len(c.HostConfig.CapDrop))
		}

		if c.HostConfig.CapDrop[0] != "SYS_ADMIN" {
			return fmt.Errorf("Container has wrong CapDrop setting: %v", c.HostConfig.CapDrop[0])
		}

		if c.HostConfig.CPUShares != 32 {
			return fmt.Errorf("Container has wrong cpu shares setting: %d", c.HostConfig.CPUShares)
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

		if _, ok := c.NetworkSettings.Networks["test"]; !ok {
			return fmt.Errorf("Container is not connected to the right user defined network: test")
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

func TestAccDockerContainer_upload(t *testing.T) {
	var c dc.Container

	testCheck := func(*terraform.State) error {
		client := testAccProvider.Meta().(*dc.Client)

		buf := new(bytes.Buffer)
		opts := dc.DownloadFromContainerOptions{
			OutputStream: buf,
			Path:         "/terraform/test.txt",
		}

		if err := client.DownloadFromContainer(c.ID, opts); err != nil {
			return fmt.Errorf("Unable to download a file from container: %s", err)
		}

		r := bytes.NewReader(buf.Bytes())
		tr := tar.NewReader(r)

		if _, err := tr.Next(); err != nil {
			return fmt.Errorf("Unable to read content of tar archive: %s", err)
		}

		fbuf := new(bytes.Buffer)
		fbuf.ReadFrom(tr)
		content := fbuf.String()

		if content != "foo" {
			return fmt.Errorf("file content is invalid")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerUploadConfig,
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
	destroy_grace_seconds = 10
	max_retry_count = 5
	memory = 512
	memory_swap = 2048
	cpu_shares = 32

	capabilities {
		add= ["ALL"]
		drop = ["SYS_ADMIN"]
	}

	dns = ["8.8.8.8"]
	dns_opts = ["rotate"]
	dns_search = ["example.com"]
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

	networks = ["${docker_network.test_network.name}"]
	network_alias = ["tftest"]

	host {
		host = "testhost"
		ip = "10.0.1.0"
	}

	host {
		host = "testhost2"
		ip = "10.0.2.0"
	}
}

resource "docker_network" "test_network" {
  name = "test"
}
`

const testAccDockerContainerUploadConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	upload {
		content = "foo"
		file = "/terraform/test.txt"
	}
}
`
