package vcd

import (
	"fmt"
	"net/url"

	"github.com/hmrc/vmware-govcd"
)

type Config struct {
	User            string
	Password        string
	Org             string
	Href            string
	VDC             string
	MaxRetryTimeout int
}

type VCDClient struct {
	*govcd.VCDClient
	MaxRetryTimeout int
}

func (c *Config) Client() (*VCDClient, error) {
	u, err := url.ParseRequestURI(c.Href)
	if err != nil {
		return nil, fmt.Errorf("Something went wrong: %s", err)
	}

	vcdclient := &VCDClient{
		govcd.NewVCDClient(*u),
		c.MaxRetryTimeout}
	org, vcd, err := vcdclient.Authenticate(c.User, c.Password, c.Org, c.VDC)
	if err != nil {
		return nil, fmt.Errorf("Something went wrong: %s", err)
	}
	vcdclient.Org = org
	vcdclient.OrgVdc = vcd
	return vcdclient, nil
}
