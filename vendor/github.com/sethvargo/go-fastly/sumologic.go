package fastly

import (
	"fmt"
	"sort"
	"time"
)

// Sumologic represents a sumologic response from the Fastly API.
type Sumologic struct {
	ServiceID string `mapstructure:"service_id"`
	Version   int    `mapstructure:"version"`

	Name              string     `mapstructure:"name"`
	Address           string     `mapstructure:"address"`
	URL               string     `mapstructure:"url"`
	Format            string     `mapstructure:"format"`
	ResponseCondition string     `mapstructure:"response_condition"`
	MessageType       string     `mapstructure:"message_type"`
	FormatVersion     int        `mapstructure:"format_version"`
	CreatedAt         *time.Time `mapstructure:"created_at"`
	UpdatedAt         *time.Time `mapstructure:"updated_at"`
	DeletedAt         *time.Time `mapstructure:"deleted_at"`
}

// sumologicsByName is a sortable list of sumologics.
type sumologicsByName []*Sumologic

// Len, Swap, and Less implement the sortable interface.
func (s sumologicsByName) Len() int      { return len(s) }
func (s sumologicsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sumologicsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListSumologicsInput is used as input to the ListSumologics function.
type ListSumologicsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version int
}

// ListSumologics returns the list of sumologics for the configuration version.
func (c *Client) ListSumologics(i *ListSumologicsInput) ([]*Sumologic, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/sumologic", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var ss []*Sumologic
	if err := decodeJSON(&ss, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(sumologicsByName(ss))
	return ss, nil
}

// CreateSumologicInput is used as input to the CreateSumologic function.
type CreateSumologicInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	Name              string `form:"name,omitempty"`
	Address           string `form:"address,omitempty"`
	URL               string `form:"url,omitempty"`
	Format            string `form:"format,omitempty"`
	ResponseCondition string `form:"response_condition,omitempty"`
	MessageType       string `form:"message_type,omitempty"`
	FormatVersion     int    `form:"format_version,omitempty"`
}

// CreateSumologic creates a new Fastly sumologic.
func (c *Client) CreateSumologic(i *CreateSumologicInput) (*Sumologic, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/sumologic", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var s *Sumologic
	if err := decodeJSON(&s, resp.Body); err != nil {
		return nil, err
	}
	return s, nil
}

// GetSumologicInput is used as input to the GetSumologic function.
type GetSumologicInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the sumologic to fetch.
	Name string
}

// GetSumologic gets the sumologic configuration with the given parameters.
func (c *Client) GetSumologic(i *GetSumologicInput) (*Sumologic, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/sumologic/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var s *Sumologic
	if err := decodeJSON(&s, resp.Body); err != nil {
		return nil, err
	}
	return s, nil
}

// UpdateSumologicInput is used as input to the UpdateSumologic function.
type UpdateSumologicInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the sumologic to update.
	Name string

	NewName           string `form:"name,omitempty"`
	Address           string `form:"address,omitempty"`
	URL               string `form:"url,omitempty"`
	Format            string `form:"format,omitempty"`
	ResponseCondition string `form:"response_condition,omitempty"`
	MessageType       string `form:"message_type,omitempty"`
	FormatVersion     int    `form:"format_version,omitempty"`
}

// UpdateSumologic updates a specific sumologic.
func (c *Client) UpdateSumologic(i *UpdateSumologicInput) (*Sumologic, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/sumologic/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var s *Sumologic
	if err := decodeJSON(&s, resp.Body); err != nil {
		return nil, err
	}
	return s, nil
}

// DeleteSumologicInput is the input parameter to DeleteSumologic.
type DeleteSumologicInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	// Name is the name of the sumologic to delete (required).
	Name string
}

// DeleteSumologic deletes the given sumologic version.
func (c *Client) DeleteSumologic(i *DeleteSumologicInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == 0 {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%d/logging/sumologic/%s", i.Service, i.Version, i.Name)
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
