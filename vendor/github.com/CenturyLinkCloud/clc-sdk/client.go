package clc

import (
	"github.com/CenturyLinkCloud/clc-sdk/aa"
	"github.com/CenturyLinkCloud/clc-sdk/alert"
	"github.com/CenturyLinkCloud/clc-sdk/api"
	"github.com/CenturyLinkCloud/clc-sdk/dc"
	"github.com/CenturyLinkCloud/clc-sdk/group"
	"github.com/CenturyLinkCloud/clc-sdk/lb"
	"github.com/CenturyLinkCloud/clc-sdk/server"
	"github.com/CenturyLinkCloud/clc-sdk/status"
)

type Client struct {
	client *api.Client

	Server *server.Service
	Status *status.Service
	AA     *aa.Service
	Alert  *alert.Service
	LB     *lb.Service
	Group  *group.Service
	DC     *dc.Service
}

func New(config api.Config) *Client {
	c := &Client{
		client: api.New(config),
	}

	c.Server = server.New(c.client)
	c.Status = status.New(c.client)
	c.AA = aa.New(c.client)
	c.Alert = alert.New(c.client)
	c.LB = lb.New(c.client)
	c.Group = group.New(c.client)
	c.DC = dc.New(c.client)

	return c
}

func (c *Client) Alias(alias string) *Client {
	c.client.Config().Alias = alias
	return c
}

func (c *Client) Authenticate() error {
	return c.client.Auth()
}
