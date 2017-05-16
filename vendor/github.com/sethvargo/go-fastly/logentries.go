package fastly

import (
	"fmt"
	"sort"
	"time"
)

// Logentries represents a logentries response from the Fastly API.
type Logentries struct {
	ServiceID string `mapstructure:"service_id"`
	Version   int    `mapstructure:"version"`

	Name              string     `mapstructure:"name"`
	Port              uint       `mapstructure:"port"`
	UseTLS            bool       `mapstructure:"use_tls"`
	Token             string     `mapstructure:"token"`
	Format            string     `mapstructure:"format"`
	ResponseCondition string     `mapstructure:"response_condition"`
	CreatedAt         *time.Time `mapstructure:"created_at"`
	UpdatedAt         *time.Time `mapstructure:"updated_at"`
	DeletedAt         *time.Time `mapstructure:"deleted_at"`
}

// logentriesByName is a sortable list of logentries.
type logentriesByName []*Logentries

// Len, Swap, and Less implement the sortable interface.
func (s logentriesByName) Len() int      { return len(s) }
func (s logentriesByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s logentriesByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListLogentriesInput is used as input to the ListLogentries function.
type ListLogentriesInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version int
}

// ListLogentries returns the list of logentries for the configuration version.
func (c *Client) ListLogentries(i *ListLogentriesInput) ([]*Logentries, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/logentries", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var ls []*Logentries
	if err := decodeJSON(&ls, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(logentriesByName(ls))
	return ls, nil
}

// CreateLogentriesInput is used as input to the CreateLogentries function.
type CreateLogentriesInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	Name              string       `form:"name,omitempty"`
	Port              uint         `form:"port,omitempty"`
	UseTLS            *Compatibool `form:"use_tls,omitempty"`
	Token             string       `form:"token,omitempty"`
	Format            string       `form:"format,omitempty"`
	ResponseCondition string       `form:"response_condition,omitempty"`
}

// CreateLogentries creates a new Fastly logentries.
func (c *Client) CreateLogentries(i *CreateLogentriesInput) (*Logentries, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/logentries", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var l *Logentries
	if err := decodeJSON(&l, resp.Body); err != nil {
		return nil, err
	}
	return l, nil
}

// GetLogentriesInput is used as input to the GetLogentries function.
type GetLogentriesInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the logentries to fetch.
	Name string
}

// GetLogentries gets the logentries configuration with the given parameters.
func (c *Client) GetLogentries(i *GetLogentriesInput) (*Logentries, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/logentries/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var l *Logentries
	if err := decodeJSON(&l, resp.Body); err != nil {
		return nil, err
	}
	return l, nil
}

// UpdateLogentriesInput is used as input to the UpdateLogentries function.
type UpdateLogentriesInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the logentries to update.
	Name string

	NewName           string       `form:"name,omitempty"`
	Port              uint         `form:"port,omitempty"`
	UseTLS            *Compatibool `form:"use_tls,omitempty"`
	Token             string       `form:"token,omitempty"`
	Format            string       `form:"format,omitempty"`
	ResponseCondition string       `form:"response_condition,omitempty"`
}

// UpdateLogentries updates a specific logentries.
func (c *Client) UpdateLogentries(i *UpdateLogentriesInput) (*Logentries, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/logentries/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var l *Logentries
	if err := decodeJSON(&l, resp.Body); err != nil {
		return nil, err
	}
	return l, nil
}

// DeleteLogentriesInput is the input parameter to DeleteLogentries.
type DeleteLogentriesInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the logentries to delete (required).
	Name string
}

// DeleteLogentries deletes the given logentries version.
func (c *Client) DeleteLogentries(i *DeleteLogentriesInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == 0 {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/logentries/%s", i.Service, i.Version, i.Name)
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
