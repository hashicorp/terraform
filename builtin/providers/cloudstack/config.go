package cloudstack

import "github.com/xanzy/go-cloudstack/cloudstack"

// Config is the configuration structure used to instantiate a
// new CloudStack client.
type Config struct {
	APIURL      string
	APIKey      string
	SecretKey   string
	HTTPGETOnly bool
	Timeout     int64
}

// NewClient returns a new CloudStack client.
func (c *Config) NewClient() (*cloudstack.CloudStackClient, error) {
	cs := cloudstack.NewAsyncClient(c.APIURL, c.APIKey, c.SecretKey, false)
	cs.HTTPGETOnly = c.HTTPGETOnly
	cs.AsyncTimeout(c.Timeout)
	return cs, nil
}
