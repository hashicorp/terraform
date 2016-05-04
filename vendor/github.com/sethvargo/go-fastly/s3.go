package fastly

import (
	"fmt"
	"sort"
	"time"
)

// S3 represents a S3 response from the Fastly API.
type S3 struct {
	ServiceID string `mapstructure:"service_id"`
	Version   string `mapstructure:"version"`

	Name              string     `mapstructure:"name"`
	BucketName        string     `mapstructure:"bucket_name"`
	Domain            string     `mapstructure:"domain"`
	AccessKey         string     `mapstructure:"access_key"`
	SecretKey         string     `mapstructure:"secret_key"`
	Path              string     `mapstructure:"path"`
	Period            uint       `mapstructure:"period"`
	GzipLevel         uint       `mapstructure:"gzip_level"`
	Format            string     `mapstructure:"format"`
	ResponseCondition string     `mapstructure:"response_condition"`
	TimestampFormat   string     `mapstructure:"timestamp_format"`
	CreatedAt         *time.Time `mapstructure:"created_at"`
	UpdatedAt         *time.Time `mapstructure:"updated_at"`
	DeletedAt         *time.Time `mapstructure:"deleted_at"`
}

// s3sByName is a sortable list of S3s.
type s3sByName []*S3

// Len, Swap, and Less implement the sortable interface.
func (s s3sByName) Len() int      { return len(s) }
func (s s3sByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s s3sByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListS3sInput is used as input to the ListS3s function.
type ListS3sInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version string
}

// ListS3s returns the list of S3s for the configuration version.
func (c *Client) ListS3s(i *ListS3sInput) ([]*S3, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/logging/s3", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var s3s []*S3
	if err := decodeJSON(&s3s, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(s3sByName(s3s))
	return s3s, nil
}

// CreateS3Input is used as input to the CreateS3 function.
type CreateS3Input struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	Name              string `form:"name,omitempty"`
	BucketName        string `form:"bucket_name,omitempty"`
	Domain            string `form:"domain,omitempty"`
	AccessKey         string `form:"access_key,omitempty"`
	SecretKey         string `form:"secret_key,omitempty"`
	Path              string `form:"path,omitempty"`
	Period            uint   `form:"period,omitempty"`
	GzipLevel         uint   `form:"gzip_level,omitempty"`
	Format            string `form:"format,omitempty"`
	ResponseCondition string `form:"response_condition,omitempty"`
	TimestampFormat   string `form:"timestamp_format,omitempty"`
}

// CreateS3 creates a new Fastly S3.
func (c *Client) CreateS3(i *CreateS3Input) (*S3, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/logging/s3", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var s3 *S3
	if err := decodeJSON(&s3, resp.Body); err != nil {
		return nil, err
	}
	return s3, nil
}

// GetS3Input is used as input to the GetS3 function.
type GetS3Input struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the S3 to fetch.
	Name string
}

// GetS3 gets the S3 configuration with the given parameters.
func (c *Client) GetS3(i *GetS3Input) (*S3, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/logging/s3/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var s3 *S3
	if err := decodeJSON(&s3, resp.Body); err != nil {
		return nil, err
	}
	return s3, nil
}

// UpdateS3Input is used as input to the UpdateS3 function.
type UpdateS3Input struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the S3 to update.
	Name string

	NewName           string `form:"name,omitempty"`
	BucketName        string `form:"bucket_name,omitempty"`
	Domain            string `form:"domain,omitempty"`
	AccessKey         string `form:"access_key,omitempty"`
	SecretKey         string `form:"secret_key,omitempty"`
	Path              string `form:"path,omitempty"`
	Period            uint   `form:"period,omitempty"`
	GzipLevel         uint   `form:"gzip_level,omitempty"`
	Format            string `form:"format,omitempty"`
	ResponseCondition string `form:"response_condition,omitempty"`
	TimestampFormat   string `form:"timestamp_format,omitempty"`
}

// UpdateS3 updates a specific S3.
func (c *Client) UpdateS3(i *UpdateS3Input) (*S3, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/logging/s3/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var s3 *S3
	if err := decodeJSON(&s3, resp.Body); err != nil {
		return nil, err
	}
	return s3, nil
}

// DeleteS3Input is the input parameter to DeleteS3.
type DeleteS3Input struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the S3 to delete (required).
	Name string
}

// DeleteS3 deletes the given S3 version.
func (c *Client) DeleteS3(i *DeleteS3Input) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == "" {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/logging/s3/%s", i.Service, i.Version, i.Name)
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
