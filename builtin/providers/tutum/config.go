package tutum

import (
	"github.com/tutumcloud/go-tutum/tutum"
)

type Config struct {
	User   string
	ApiKey string
}

func (c *Config) Load() error {
	tutum.User = c.User
	tutum.ApiKey = c.ApiKey
	return tutum.LoadAuth()
}
