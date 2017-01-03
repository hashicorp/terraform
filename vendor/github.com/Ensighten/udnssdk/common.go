package udnssdk

import "net/http"

// GetResultByURI just requests a URI
func (c *Client) GetResultByURI(uri string) (*http.Response, error) {
	req, err := c.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTPClient.Do(req)

	if err != nil {
		return res, err
	}
	return res, err
}
