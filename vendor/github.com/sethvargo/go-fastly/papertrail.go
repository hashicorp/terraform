package fastly

import (
	"fmt"
	"sort"
	"time"
)

// Papertrail represents a papertrail response from the Fastly API.
type Papertrail struct {
	ServiceID string `mapstructure:"service_id"`
	Version   int    `mapstructure:"version"`

	Name              string     `mapstructure:"name"`
	Address           string     `mapstructure:"address"`
	Port              uint       `mapstructure:"port"`
	Format            string     `mapstructure:"format"`
	ResponseCondition string     `mapstructure:"response_condition"`
	CreatedAt         *time.Time `mapstructure:"created_at"`
	UpdatedAt         *time.Time `mapstructure:"updated_at"`
	DeletedAt         *time.Time `mapstructure:"deleted_at"`
}

// papertrailsByName is a sortable list of papertrails.
type papertrailsByName []*Papertrail

// Len, Swap, and Less implement the sortable interface.
func (s papertrailsByName) Len() int      { return len(s) }
func (s papertrailsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s papertrailsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListPapertrailsInput is used as input to the ListPapertrails function.
type ListPapertrailsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version int
}

// ListPapertrails returns the list of papertrails for the configuration version.
func (c *Client) ListPapertrails(i *ListPapertrailsInput) ([]*Papertrail, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/papertrail", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var ps []*Papertrail
	if err := decodeJSON(&ps, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(papertrailsByName(ps))
	return ps, nil
}

// CreatePapertrailInput is used as input to the CreatePapertrail function.
type CreatePapertrailInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	Name              string     `form:"name,omitempty"`
	Address           string     `form:"address,omitempty"`
	Port              uint       `form:"port,omitempty"`
	Format            string     `form:"format,omitempty"`
	ResponseCondition string     `form:"response_condition,omitempty"`
	CreatedAt         *time.Time `form:"created_at,omitempty"`
	UpdatedAt         *time.Time `form:"updated_at,omitempty"`
	DeletedAt         *time.Time `form:"deleted_at,omitempty"`
}

// CreatePapertrail creates a new Fastly papertrail.
func (c *Client) CreatePapertrail(i *CreatePapertrailInput) (*Papertrail, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/papertrail", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var p *Papertrail
	if err := decodeJSON(&p, resp.Body); err != nil {
		return nil, err
	}
	return p, nil
}

// GetPapertrailInput is used as input to the GetPapertrail function.
type GetPapertrailInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the papertrail to fetch.
	Name string
}

// GetPapertrail gets the papertrail configuration with the given parameters.
func (c *Client) GetPapertrail(i *GetPapertrailInput) (*Papertrail, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/papertrail/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var p *Papertrail
	if err := decodeJSON(&p, resp.Body); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdatePapertrailInput is used as input to the UpdatePapertrail function.
type UpdatePapertrailInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the papertrail to update.
	Name string

	NewName           string     `form:"name,omitempty"`
	Address           string     `form:"address,omitempty"`
	Port              uint       `form:"port,omitempty"`
	Format            string     `form:"format,omitempty"`
	ResponseCondition string     `form:"response_condition,omitempty"`
	CreatedAt         *time.Time `form:"created_at,omitempty"`
	UpdatedAt         *time.Time `form:"updated_at,omitempty"`
	DeletedAt         *time.Time `form:"deleted_at,omitempty"`
}

// UpdatePapertrail updates a specific papertrail.
func (c *Client) UpdatePapertrail(i *UpdatePapertrailInput) (*Papertrail, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/papertrail/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var p *Papertrail
	if err := decodeJSON(&p, resp.Body); err != nil {
		return nil, err
	}
	return p, nil
}

// DeletePapertrailInput is the input parameter to DeletePapertrail.
type DeletePapertrailInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the papertrail to delete (required).
	Name string
}

// DeletePapertrail deletes the given papertrail version.
func (c *Client) DeletePapertrail(i *DeletePapertrailInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == 0 {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/papertrail/%s", i.Service, i.Version, i.Name)
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
