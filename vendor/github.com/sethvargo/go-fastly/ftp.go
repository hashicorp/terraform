package fastly

import (
	"fmt"
	"sort"
	"time"
)

// FTP represents an FTP logging response from the Fastly API.
type FTP struct {
	ServiceID string `mapstructure:"service_id"`
	Version   int    `mapstructure:"version"`

	Name              string     `mapstructure:"name"`
	Address           string     `mapstructure:"address"`
	Port              uint       `mapstructure:"port"`
	Username          string     `mapstructure:"user"`
	Password          string     `mapstructure:"password"`
	Path              string     `mapstructure:"path"`
	Period            uint       `mapstructure:"period"`
	GzipLevel         uint8      `mapstructure:"gzip_level"`
	Format            string     `mapstructure:"format"`
	ResponseCondition string     `mapstructure:"response_condition"`
	TimestampFormat   string     `mapstructure:"timestamp_format"`
	CreatedAt         *time.Time `mapstructure:"created_at"`
	UpdatedAt         *time.Time `mapstructure:"updated_at"`
	DeletedAt         *time.Time `mapstructure:"deleted_at"`
}

// ftpsByName is a sortable list of ftps.
type ftpsByName []*FTP

// Len, Swap, and Less implement the sortable interface.
func (s ftpsByName) Len() int      { return len(s) }
func (s ftpsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ftpsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListFTPsInput is used as input to the ListFTPs function.
type ListFTPsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version int
}

// ListFTPs returns the list of ftps for the configuration version.
func (c *Client) ListFTPs(i *ListFTPsInput) ([]*FTP, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/ftp", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var ftps []*FTP
	if err := decodeJSON(&ftps, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(ftpsByName(ftps))
	return ftps, nil
}

// CreateFTPInput is used as input to the CreateFTP function.
type CreateFTPInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	Name              string `form:"name,omitempty"`
	Address           string `form:"address,omitempty"`
	Port              uint   `form:"port,omitempty"`
	Username          string `form:"user,omitempty"`
	Password          string `form:"password,omitempty"`
	Path              string `form:"path,omitempty"`
	Period            uint   `form:"period,omitempty"`
	GzipLevel         uint8  `form:"gzip_level,omitempty"`
	Format            string `form:"format,omitempty"`
	ResponseCondition string `form:"response_condition,omitempty"`
	TimestampFormat   string `form:"timestamp_format,omitempty"`
}

// CreateFTP creates a new Fastly FTP.
func (c *Client) CreateFTP(i *CreateFTPInput) (*FTP, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/ftp", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var ftp *FTP
	if err := decodeJSON(&ftp, resp.Body); err != nil {
		return nil, err
	}
	return ftp, nil
}

// GetFTPInput is used as input to the GetFTP function.
type GetFTPInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the FTP to fetch.
	Name string
}

// GetFTP gets the FTP configuration with the given parameters.
func (c *Client) GetFTP(i *GetFTPInput) (*FTP, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/ftp/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var b *FTP
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateFTPInput is used as input to the UpdateFTP function.
type UpdateFTPInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the FTP to update.
	Name string

	NewName           string `form:"name,omitempty"`
	Address           string `form:"address,omitempty"`
	Port              uint   `form:"port,omitempty"`
	Username          string `form:"user,omitempty"`
	Password          string `form:"password,omitempty"`
	Path              string `form:"path,omitempty"`
	Period            uint   `form:"period,omitempty"`
	GzipLevel         uint8  `form:"gzip_level,omitempty"`
	Format            string `form:"format,omitempty"`
	ResponseCondition string `form:"response_condition,omitempty"`
	TimestampFormat   string `form:"timestamp_format,omitempty"`
}

// UpdateFTP updates a specific FTP.
func (c *Client) UpdateFTP(i *UpdateFTPInput) (*FTP, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/ftp/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *FTP
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteFTPInput is the input parameter to DeleteFTP.
type DeleteFTPInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the FTP to delete (required).
	Name string
}

// DeleteFTP deletes the given FTP version.
func (c *Client) DeleteFTP(i *DeleteFTPInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == 0 {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/ftp/%s", i.Service, i.Version, i.Name)
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
