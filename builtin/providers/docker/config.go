package docker

import (
	"path/filepath"

	dc "github.com/fsouza/go-dockerclient"
)

// Config is the structure that stores the configuration to talk to a
// Docker API compatible host.
type Config struct {
	Host     string
	CertPath string
}

// NewClient() returns a new Docker client.
func (c *Config) NewClient() (*dc.Client, error) {
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
