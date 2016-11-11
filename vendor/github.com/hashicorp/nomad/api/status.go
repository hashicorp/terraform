package api

// Status is used to query the status-related endpoints.
type Status struct {
	client *Client
}

// Status returns a handle on the status endpoints.
func (c *Client) Status() *Status {
	return &Status{client: c}
}

// Leader is used to query for the current cluster leader.
func (s *Status) Leader() (string, error) {
	var resp string
	_, err := s.client.query("/v1/status/leader", &resp, nil)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// RegionLeader is used to query for the leader in the passed region.
func (s *Status) RegionLeader(region string) (string, error) {
	var resp string
	q := QueryOptions{Region: region}
	_, err := s.client.query("/v1/status/leader", &resp, &q)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// Peers is used to query the addresses of the server peers
// in the cluster.
func (s *Status) Peers() ([]string, error) {
	var resp []string
	_, err := s.client.query("/v1/status/peers", &resp, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
