package udnssdk

// GetResultByURI just requests a URI
func (c *Client) GetResultByURI(uri string) (*Response, error) {
	req, err := c.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTPClient.Do(req)

	if err != nil {
		return &Response{Response: res}, err
	}
	return &Response{Response: res}, err
}
