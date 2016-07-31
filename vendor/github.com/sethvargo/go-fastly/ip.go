package fastly

// IPAddrs is a sortable list of IP addresses returned by the Fastly API.
type IPAddrs []string

// IPs returns the list of public IP addresses for Fastly's network.
func (c *Client) IPs() (IPAddrs, error) {
	resp, err := c.Get("/public-ip-list", nil)
	if err != nil {
		return nil, err
	}

	var m map[string][]string
	if err := decodeJSON(&m, resp.Body); err != nil {
		return nil, err
	}
	return IPAddrs(m["addresses"]), nil
}
