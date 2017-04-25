package fastly

import (
	"fmt"
	"sort"
)

// Domain represents the the domain name Fastly will serve content for.
type Domain struct {
	ServiceID string `mapstructure:"service_id"`
	Version   int    `mapstructure:"version"`

	Name    string `mapstructure:"name"`
	Comment string `mapstructure:"comment"`
	Locked  bool   `mapstructure:"locked"`
}

// domainsByName is a sortable list of backends.
type domainsByName []*Domain

// Len, Swap, and Less implement the sortable interface.
func (s domainsByName) Len() int      { return len(s) }
func (s domainsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s domainsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListDomainsInput is used as input to the ListDomains function.
type ListDomainsInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int
}

// ListDomains returns the list of domains for this Service.
func (c *Client) ListDomains(i *ListDomainsInput) ([]*Domain, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/domain", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var ds []*Domain
	if err := decodeJSON(&ds, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(domainsByName(ds))
	return ds, nil
}

// CreateDomainInput is used as input to the CreateDomain function.
type CreateDomainInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the domain that the service will respond to (required).
	Name string `form:"name"`

	// Comment is a personal, freeform descriptive note.
	Comment string `form:"comment,omitempty"`
}

// CreateDomain creates a new domain with the given information.
func (c *Client) CreateDomain(i *CreateDomainInput) (*Domain, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/domain", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var d *Domain
	if err := decodeJSON(&d, resp.Body); err != nil {
		return nil, err
	}
	return d, nil
}

// GetDomainInput is used as input to the GetDomain function.
type GetDomainInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the domain to fetch.
	Name string `form:"name"`
}

// GetDomain retrieves information about the given domain name.
func (c *Client) GetDomain(i *GetDomainInput) (*Domain, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/domain/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var d *Domain
	if err := decodeJSON(&d, resp.Body); err != nil {
		return nil, err
	}
	return d, nil
}

// UpdateDomainInput is used as input to the UpdateDomain function.
type UpdateDomainInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the domain that the service will respond to (required).
	Name string

	// NewName is the updated name of the domain
	NewName string `form:"name"`

	// Comment is a personal, freeform descriptive note.
	Comment string `form:"comment,omitempty"`
}

// UpdateDomain updates a single domain for the current service. The only allowed
// parameters are `Name` and `Comment`.
func (c *Client) UpdateDomain(i *UpdateDomainInput) (*Domain, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/domain/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var d *Domain
	if err := decodeJSON(&d, resp.Body); err != nil {
		return nil, err
	}
	return d, nil
}

// DeleteDomainInput is used as input to the DeleteDomain function.
type DeleteDomainInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the domain that the service will respond to (required).
	Name string `form:"name"`
}

// DeleteDomain removes a single domain by the given name.
func (c *Client) DeleteDomain(i *DeleteDomainInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == 0 {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/domain/%s", i.Service, i.Version, i.Name)
	_, err := c.Delete(path, nil)
	if err != nil {
		return err
	}
	return nil
}
