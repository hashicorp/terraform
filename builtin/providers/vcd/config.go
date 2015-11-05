package vcd

import (
	"fmt"
	"net/url"

	"github.com/opencredo/vmware-govcd"
)

type Config struct {
	User     string
	Password string
	Org      string
	Href     string
	VDC      string
}

func (c *Config) Client() (*govcd.VCDClient, error) {
	u, err := url.ParseRequestURI(c.Href)
	if err != nil {
		return nil, fmt.Errorf("Something went wrong: %s", err)
	}

	vcdclient := govcd.NewVCDClient(*u)
	org, vcd, err := vcdclient.Authenticate(c.User, c.Password, c.Org, c.VDC)
	if err != nil {
		return nil, fmt.Errorf("Something went wrong: %s", err)
	}
	vcdclient.Org = org
	vcdclient.OrgVdc = vcd
	return vcdclient, nil
}
