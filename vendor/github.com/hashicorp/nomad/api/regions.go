package api

import "sort"

// Regions is used to query the regions in the cluster.
type Regions struct {
	client *Client
}

// Regions returns a handle on the allocs endpoints.
func (c *Client) Regions() *Regions {
	return &Regions{client: c}
}

// List returns a list of all of the regions.
func (r *Regions) List() ([]string, error) {
	var resp []string
	if _, err := r.client.query("/v1/regions", &resp, nil); err != nil {
		return nil, err
	}
	sort.Strings(resp)
	return resp, nil
}
