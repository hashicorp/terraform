package fastly

import (
	"fmt"
	"sort"
)

const (
	// DirectorTypeRandom is a director that does random direction.
	DirectorTypeRandom DirectorType = 1

	// DirectorTypeRoundRobin is a director that does round-robin direction.
	DirectorTypeRoundRobin DirectorType = 2

	// DirectorTypeHash is a director that does hash direction.
	DirectorTypeHash DirectorType = 3

	// DirectorTypeClient is a director that does client direction.
	DirectorTypeClient DirectorType = 4
)

// DirectorType is a type of director.
type DirectorType uint8

// Director represents a director response from the Fastly API.
type Director struct {
	ServiceID string `mapstructure:"service_id"`
	Version   int    `mapstructure:"version"`

	Name     string       `mapstructure:"name"`
	Comment  string       `mapstructure:"comment"`
	Quorum   uint         `mapstructure:"quorum"`
	Type     DirectorType `mapstructure:"type"`
	Retries  uint         `mapstructure:"retries"`
	Capacity uint         `mapstructure:"capacity"`
}

// directorsByName is a sortable list of directors.
type directorsByName []*Director

// Len, Swap, and Less implement the sortable interface.
func (s directorsByName) Len() int      { return len(s) }
func (s directorsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s directorsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListDirectorsInput is used as input to the ListDirectors function.
type ListDirectorsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version int
}

// ListDirectors returns the list of directors for the configuration version.
func (c *Client) ListDirectors(i *ListDirectorsInput) ([]*Director, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/director", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var ds []*Director
	if err := decodeJSON(&ds, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(directorsByName(ds))
	return ds, nil
}

// CreateDirectorInput is used as input to the CreateDirector function.
type CreateDirectorInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	Name    string       `form:"name,omitempty"`
	Comment string       `form:"comment,omitempty"`
	Quorum  uint         `form:"quorum,omitempty"`
	Type    DirectorType `form:"type,omitempty"`
	Retries uint         `form:"retries,omitempty"`
}

// CreateDirector creates a new Fastly director.
func (c *Client) CreateDirector(i *CreateDirectorInput) (*Director, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/director", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var d *Director
	if err := decodeJSON(&d, resp.Body); err != nil {
		return nil, err
	}
	return d, nil
}

// GetDirectorInput is used as input to the GetDirector function.
type GetDirectorInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the director to fetch.
	Name string
}

// GetDirector gets the director configuration with the given parameters.
func (c *Client) GetDirector(i *GetDirectorInput) (*Director, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/director/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var d *Director
	if err := decodeJSON(&d, resp.Body); err != nil {
		return nil, err
	}
	return d, nil
}

// UpdateDirectorInput is used as input to the UpdateDirector function.
type UpdateDirectorInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the director to update.
	Name string

	Comment string       `form:"comment,omitempty"`
	Quorum  uint         `form:"quorum,omitempty"`
	Type    DirectorType `form:"type,omitempty"`
	Retries uint         `form:"retries,omitempty"`
}

// UpdateDirector updates a specific director.
func (c *Client) UpdateDirector(i *UpdateDirectorInput) (*Director, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/director/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var d *Director
	if err := decodeJSON(&d, resp.Body); err != nil {
		return nil, err
	}
	return d, nil
}

// DeleteDirectorInput is the input parameter to DeleteDirector.
type DeleteDirectorInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the director to delete (required).
	Name string
}

// DeleteDirector deletes the given director version.
func (c *Client) DeleteDirector(i *DeleteDirectorInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == 0 {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/director/%s", i.Service, i.Version, i.Name)
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
