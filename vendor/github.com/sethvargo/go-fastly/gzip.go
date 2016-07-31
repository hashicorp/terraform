package fastly

import (
	"fmt"
	"sort"
)

// Gzip represents an Gzip logging response from the Fastly API.
type Gzip struct {
	ServiceID string `mapstructure:"service_id"`
	Version   string `mapstructure:"version"`

	Name           string `mapstructure:"name"`
	ContentTypes   string `mapstructure:"content_types"`
	Extensions     string `mapstructure:"extensions"`
	CacheCondition string `mapstructure:"cache_condition"`
}

// gzipsByName is a sortable list of gzips.
type gzipsByName []*Gzip

// Len, Swap, and Less implement the sortable interface.
func (s gzipsByName) Len() int      { return len(s) }
func (s gzipsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s gzipsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListGzipsInput is used as input to the ListGzips function.
type ListGzipsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version string
}

// ListGzips returns the list of gzips for the configuration version.
func (c *Client) ListGzips(i *ListGzipsInput) ([]*Gzip, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/gzip", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var gzips []*Gzip
	if err := decodeJSON(&gzips, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(gzipsByName(gzips))
	return gzips, nil
}

// CreateGzipInput is used as input to the CreateGzip function.
type CreateGzipInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	Name           string `form:"name,omitempty"`
	ContentTypes   string `form:"content_types"`
	Extensions     string `form:"extensions"`
	CacheCondition string `form:"cache_condition,omitempty"`
}

// CreateGzip creates a new Fastly Gzip.
func (c *Client) CreateGzip(i *CreateGzipInput) (*Gzip, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/gzip", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var gzip *Gzip
	if err := decodeJSON(&gzip, resp.Body); err != nil {
		return nil, err
	}
	return gzip, nil
}

// GetGzipInput is used as input to the GetGzip function.
type GetGzipInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the Gzip to fetch.
	Name string
}

// GetGzip gets the Gzip configuration with the given parameters.
func (c *Client) GetGzip(i *GetGzipInput) (*Gzip, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/gzip/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var b *Gzip
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateGzipInput is used as input to the UpdateGzip function.
type UpdateGzipInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the Gzip to update.
	Name string

	NewName        string `form:"name,omitempty"`
	ContentTypes   string `form:"content_types,omitempty"`
	Extensions     string `form:"extensions,omitempty"`
	CacheCondition string `form:"cache_condition,omitempty"`
}

// UpdateGzip updates a specific Gzip.
func (c *Client) UpdateGzip(i *UpdateGzipInput) (*Gzip, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/gzip/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *Gzip
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteGzipInput is the input parameter to DeleteGzip.
type DeleteGzipInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the Gzip to delete (required).
	Name string
}

// DeleteGzip deletes the given Gzip version.
func (c *Client) DeleteGzip(i *DeleteGzipInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == "" {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/gzip/%s", i.Service, i.Version, i.Name)
	resp, err := c.Delete(path, nil)
	if err != nil {
		return err
	}

	var r *statusResp
	if err := decodeJSON(&r, resp.Body); err != nil {
		return err
	}
	if !r.Ok() {
		return fmt.Errorf("Not Ok")
	}
	return nil
}
