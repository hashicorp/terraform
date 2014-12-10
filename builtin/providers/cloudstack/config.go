package cloudstack

import "github.com/xanzy/go-cloudstack/cloudstack"

// Config is the configuration structure used to instantiate a
// new CloudStack client.
type Config struct {
	ApiURL    string
	ApiKey    string
	SecretKey string
}

// Client() returns a new CloudStack client.
func (c *Config) NewClient() (*cloudstack.CloudStackClient, error) {
	cs := cloudstack.NewAsyncClient(c.ApiURL, c.ApiKey, c.SecretKey, false)
	cs.AsyncTimeout(180)
	return cs, nil
}
