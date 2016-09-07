package vcd

import (
	"fmt"
	"net/url"

	"github.com/vmware/govcloudair"
)

type Config struct {
	User            string
	Password        string
	Org             string
	Href            string
	VDC             string
	MaxRetryTimeout int
	InsecureFlag    bool
}

type VCDClient struct {
	*govcloudair.VCDClient
	MaxRetryTimeout int
	InsecureFlag    bool
}

func (c *Config) Client() (*VCDClient, error) {
	u, err := url.ParseRequestURI(c.Href)
	if err != nil {
		return nil, fmt.Errorf("Something went wrong: %s", err)
	}

	vcdclient := &VCDClient{
		govcloudair.NewVCDClient(*u, c.InsecureFlag),
		c.MaxRetryTimeout, c.InsecureFlag}
	org, vcd, err := vcdclient.Authenticate(c.User, c.Password, c.Org, c.VDC)
	if err != nil {
		return nil, fmt.Errorf("Something went wrong: %s", err)
	}
	vcdclient.Org = org
	vcdclient.OrgVdc = vcd
	return vcdclient, nil
}
