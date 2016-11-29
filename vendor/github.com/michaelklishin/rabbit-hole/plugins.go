package rabbithole

func (c *Client) EnabledProtocols() (xs []string, err error) {
	overview, err := c.Overview()
	if err != nil {
		return []string{}, err
	}

	// we really need to implement Map/Filter/etc. MK.
	xs = make([]string, len(overview.Listeners))
	for i, lnr := range overview.Listeners {
		xs[i] = lnr.Protocol
	}

	return xs, nil
}

func (c *Client) ProtocolPorts() (res map[string]Port, err error) {
	res = map[string]Port{}

	overview, err := c.Overview()
	if err != nil {
		return res, err
	}

	for _, lnr := range overview.Listeners {
		res[lnr.Protocol] = lnr.Port
	}

	return res, nil
}
