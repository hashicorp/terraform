package fastly

import "fmt"

// Purge is a response from a purge request.
type Purge struct {
	// Status is the status of the purge, usually "ok".
	Status string `mapstructure:"status"`

	// ID is the unique ID of the purge request.
	ID string `mapstructure:"id"`
}

// PurgeInput is used as input to the Purge function.
type PurgeInput struct {
	// URL is the URL to purge (required).
	URL string

	// Soft performs a soft purge.
	Soft bool
}

// Purge instantly purges an individual URL.
func (c *Client) Purge(i *PurgeInput) (*Purge, error) {
	if i.URL == "" {
		return nil, ErrMissingURL
	}

	req, err := c.RawRequest("PURGE", i.URL, nil)
	if err != nil {
		return nil, err
	}

	if i.Soft {
		req.Header.Set("Fastly-Soft-Purge", "1")
	}

	resp, err := checkResp(c.HTTPClient.Do(req))
	if err != nil {
		return nil, err
	}

	var r *Purge
	if err := decodeJSON(&r, resp.Body); err != nil {
		return nil, err
	}
	return r, nil
}

// PurgeKeyInput is used as input to the Purge function.
type PurgeKeyInput struct {
	// Service is the ID of the service (required).
	Service string

	// Key is the key to purge (required).
	Key string

	// Soft performs a soft purge.
	Soft bool
}

// PurgeKey instantly purges a particular service of items tagged with a key.
func (c *Client) PurgeKey(i *PurgeKeyInput) (*Purge, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Key == "" {
		return nil, ErrMissingKey
	}

	path := fmt.Sprintf("/service/%s/purge/%s", i.Service, i.Key)
	req, err := c.RawRequest("POST", path, nil)
	if err != nil {
		return nil, err
	}

	if i.Soft {
		req.Header.Set("Fastly-Soft-Purge", "1")
	}

	resp, err := checkResp(c.HTTPClient.Do(req))
	if err != nil {
		return nil, err
	}

	var r *Purge
	if err := decodeJSON(&r, resp.Body); err != nil {
		return nil, err
	}
	return r, nil
}

// PurgeAllInput is used as input to the Purge function.
type PurgeAllInput struct {
	// Service is the ID of the service (required).
	Service string

	// Soft performs a soft purge.
	Soft bool
}

// PurgeAll instantly purges everything from a service.
func (c *Client) PurgeAll(i *PurgeAllInput) (*Purge, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	path := fmt.Sprintf("/service/%s/purge_all", i.Service)
	req, err := c.RawRequest("POST", path, nil)
	if err != nil {
		return nil, err
	}

	if i.Soft {
		req.Header.Set("Fastly-Soft-Purge", "1")
	}

	resp, err := checkResp(c.HTTPClient.Do(req))
	if err != nil {
		return nil, err
	}

	var r *Purge
	if err := decodeJSON(&r, resp.Body); err != nil {
		return nil, err
	}
	return r, nil

}
