package sdc

import (
	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/jpc"
	"github.com/joyent/gosdc/cloudapi"
	"github.com/joyent/gosign/auth"
)

type Config struct {
	SdcKeyName string

	creds      *auth.Credentials
	sdc_client *cloudapi.Client
}

func (c *Config) initialize() (err error) {
	if c.creds, err = jpc.CompleteCredentialsFromEnv(c.SdcKeyName); err != nil {
		return err
	}

	c.sdc_client = cloudapi.New(client.NewClient(c.creds.SdcEndpoint.URL, cloudapi.DefaultAPIVersion, c.creds, &cloudapi.Logger))

	return nil
}
