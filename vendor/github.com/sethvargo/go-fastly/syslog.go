package fastly

import (
	"fmt"
	"sort"
	"time"
)

// Syslog represents a syslog response from the Fastly API.
type Syslog struct {
	ServiceID string `mapstructure:"service_id"`
	Version   int    `mapstructure:"version"`

	Name              string     `mapstructure:"name"`
	Address           string     `mapstructure:"address"`
	Port              uint       `mapstructure:"port"`
	UseTLS            bool       `mapstructure:"use_tls"`
	TLSCACert         string     `mapstructure:"tls_ca_cert"`
	Token             string     `mapstructure:"token"`
	Format            string     `mapstructure:"format"`
	FormatVersion     uint       `mapstructure:"format_version"`
	ResponseCondition string     `mapstructure:"response_condition"`
	CreatedAt         *time.Time `mapstructure:"created_at"`
	UpdatedAt         *time.Time `mapstructure:"updated_at"`
	DeletedAt         *time.Time `mapstructure:"deleted_at"`
}

// syslogsByName is a sortable list of syslogs.
type syslogsByName []*Syslog

// Len, Swap, and Less implement the sortable interface.
func (s syslogsByName) Len() int      { return len(s) }
func (s syslogsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s syslogsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListSyslogsInput is used as input to the ListSyslogs function.
type ListSyslogsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version int
}

// ListSyslogs returns the list of syslogs for the configuration version.
func (c *Client) ListSyslogs(i *ListSyslogsInput) ([]*Syslog, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/syslog", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var ss []*Syslog
	if err := decodeJSON(&ss, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(syslogsByName(ss))
	return ss, nil
}

// CreateSyslogInput is used as input to the CreateSyslog function.
type CreateSyslogInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	Name              string       `form:"name,omitempty"`
	Address           string       `form:"address,omitempty"`
	Port              uint         `form:"port,omitempty"`
	UseTLS            *Compatibool `form:"use_tls,omitempty"`
	TLSCACert         string       `form:"tls_ca_cert,omitempty"`
	Token             string       `form:"token,omitempty"`
	Format            string       `form:"format,omitempty"`
	FormatVersion     uint         `form:"format_version,omitempty"`
	ResponseCondition string       `form:"response_condition,omitempty"`
}

// CreateSyslog creates a new Fastly syslog.
func (c *Client) CreateSyslog(i *CreateSyslogInput) (*Syslog, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/syslog", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var s *Syslog
	if err := decodeJSON(&s, resp.Body); err != nil {
		return nil, err
	}
	return s, nil
}

// GetSyslogInput is used as input to the GetSyslog function.
type GetSyslogInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the syslog to fetch.
	Name string
}

// GetSyslog gets the syslog configuration with the given parameters.
func (c *Client) GetSyslog(i *GetSyslogInput) (*Syslog, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/syslog/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var s *Syslog
	if err := decodeJSON(&s, resp.Body); err != nil {
		return nil, err
	}
	return s, nil
}

// UpdateSyslogInput is used as input to the UpdateSyslog function.
type UpdateSyslogInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the syslog to update.
	Name string

	NewName           string       `form:"name,omitempty"`
	Address           string       `form:"address,omitempty"`
	Port              uint         `form:"port,omitempty"`
	UseTLS            *Compatibool `form:"use_tls,omitempty"`
	TLSCACert         string       `form:"tls_ca_cert,omitempty"`
	Token             string       `form:"token,omitempty"`
	Format            string       `form:"format,omitempty"`
	FormatVersion     uint         `form:"format_version,omitempty"`
	ResponseCondition string       `form:"response_condition,omitempty"`
}

// UpdateSyslog updates a specific syslog.
func (c *Client) UpdateSyslog(i *UpdateSyslogInput) (*Syslog, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/syslog/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var s *Syslog
	if err := decodeJSON(&s, resp.Body); err != nil {
		return nil, err
	}
	return s, nil
}

// DeleteSyslogInput is the input parameter to DeleteSyslog.
type DeleteSyslogInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the syslog to delete (required).
	Name string
}

// DeleteSyslog deletes the given syslog version.
func (c *Client) DeleteSyslog(i *DeleteSyslogInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == 0 {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/syslog/%s", i.Service, i.Version, i.Name)
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
