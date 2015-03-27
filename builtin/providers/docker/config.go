package docker

import dc "github.com/fsouza/go-dockerclient"

type Config struct {
	DockerHost string
	SkipPull   bool
}

type Data struct {
	DockerImages map[string]*dc.APIImages
}

// NewClient() returns a new Docker client.
func (c *Config) NewClient() (*dc.Client, error) {
	return dc.NewClient(c.DockerHost)
}

// NewData() returns a new data struct.
func (c *Config) NewData() *Data {
	return &Data{
		DockerImages: map[string]*dc.APIImages{},
	}
}
