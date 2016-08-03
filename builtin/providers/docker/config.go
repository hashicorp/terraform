package docker

import (
	"path/filepath"
	"strings"

	dc "github.com/fsouza/go-dockerclient"
)

// DockerConfig is the structure that stores the configuration to talk to a
// Docker API compatible host.
type DockerConfig struct {
	Host     string
	CertPath string
}

// NewClient() returns a new Docker client.
func (c *DockerConfig) NewClient() (*dc.Client, error) {
	// If there is no cert information, then just return the direct client
	if c.CertPath == "" {
		return dc.NewClient(c.Host)
	}

	// If there is cert information, load it and use it.
	ca := filepath.Join(c.CertPath, "ca.pem")
	cert := filepath.Join(c.CertPath, "cert.pem")
	key := filepath.Join(c.CertPath, "key.pem")
	return dc.NewTLSClient(c.Host, cert, key, ca)
}

// Data ia structure for holding data that we fetch from Docker.
type Data struct {
	DockerImages map[string]*dc.APIImages
}

type ProviderConfig struct {
	DockerClient *dc.Client
	AuthConfigs  *dc.AuthConfigurations
}

// The registry address can be referenced in various places (registry auth, docker config file, image name)
// with or without the http(s):// prefix; this function is used to standardize the inputs
func normalizeRegistryAddress(address string) string {
	if !strings.HasPrefix(address, "https://") && !strings.HasPrefix(address, "http://") {
		return "https://" + address
	}
	return address
}
