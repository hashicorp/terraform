package cloudapi

import (
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// ListDatacenters provides a list of all datacenters this cloud is aware of.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListDatacenters
func (c *Client) ListDatacenters() (map[string]interface{}, error) {
	var resp map[string]interface{}
	req := request{
		method: client.GET,
		url:    apiDatacenters,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of datcenters")
	}
	return resp, nil
}

// GetDatacenter gets an individual datacenter by name. Returns an HTTP redirect
// to your client, the datacenter URL is in the Location header.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetDatacenter
func (c *Client) GetDatacenter(datacenterName string) (string, error) {
	var respHeader http.Header
	req := request{
		method:         client.GET,
		url:            makeURL(apiDatacenters, datacenterName),
		respHeader:     &respHeader,
		expectedStatus: http.StatusFound,
	}
	respData, err := c.sendRequest(req)
	if err != nil {
		return "", errors.Newf(err, "failed to get datacenter with name: %s", datacenterName)
	}
	return respData.RespHeaders.Get("Location"), nil
}
