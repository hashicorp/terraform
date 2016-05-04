package cloudapi

import (
	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// list available services
func (c *Client) ListServices() (map[string]string, error) {
	var resp map[string]string
	req := request{
		method: client.GET,
		url:    apiServices,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of services")
	}
	return resp, nil
}
