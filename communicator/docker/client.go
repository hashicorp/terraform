package docker

import (
	"path/filepath"

	dc "github.com/fsouza/go-dockerclient"
)

// NewClient() returns a new Docker client.
func (c *connectionInfo) NewClient() (*dc.Client, error) {
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
