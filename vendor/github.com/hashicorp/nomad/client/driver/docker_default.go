//+build !windows

package driver

import docker "github.com/fsouza/go-dockerclient"

const (
	// Setting default network mode for non-windows OS as bridge
	defaultNetworkMode = "bridge"
)

func getPortBinding(ip string, port string) []docker.PortBinding {
	return []docker.PortBinding{docker.PortBinding{HostIP: ip, HostPort: port}}
}
