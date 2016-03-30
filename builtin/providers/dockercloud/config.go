package dockercloud

import (
	"github.com/docker/go-dockercloud/dockercloud"
)

type Config struct {
	User    string
	ApiKey  string
	BaseUrl string
}

func (c *Config) Load() error {
	dockercloud.User = c.User
	dockercloud.ApiKey = c.ApiKey
	dockercloud.BaseUrl = c.BaseUrl + "/api/"
	return dockercloud.LoadAuth()
}
