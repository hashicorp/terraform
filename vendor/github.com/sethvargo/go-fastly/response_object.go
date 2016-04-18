package fastly

import (
	"fmt"
	"sort"
)

// ResponseObject represents a response object response from the Fastly API.
type ResponseObject struct {
	ServiceID string `mapstructure:"service_id"`
	Version   string `mapstructure:"version"`

	Name             string `mapstructure:"name"`
	Status           uint   `mapstructure:"status"`
	Response         string `mapstructure:"response"`
	Content          string `mapstructure:"content"`
	ContentType      string `mapstructure:"content_type"`
	RequestCondition string `mapstructure:"request_condition"`
	CacheCondition   string `mapstructure:"cache_condition"`
}

// responseObjectsByName is a sortable list of response objects.
type responseObjectsByName []*ResponseObject

// Len, Swap, and Less implement the sortable interface.
func (s responseObjectsByName) Len() int      { return len(s) }
func (s responseObjectsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s responseObjectsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListResponseObjectsInput is used as input to the ListResponseObjects
// function.
type ListResponseObjectsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version string
}

// ListResponseObjects returns the list of response objects for the
// configuration version.
func (c *Client) ListResponseObjects(i *ListResponseObjectsInput) ([]*ResponseObject, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/response_object", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var bs []*ResponseObject
	if err := decodeJSON(&bs, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(responseObjectsByName(bs))
	return bs, nil
}

// CreateResponseObjectInput is used as input to the CreateResponseObject
// function.
type CreateResponseObjectInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	Name             string `form:"name,omitempty"`
	Status           uint   `form:"status,omitempty"`
	Response         string `form:"response,omitempty"`
	Content          string `form:"content,omitempty"`
	ContentType      string `form:"content_type,omitempty"`
	RequestCondition string `form:"request_condition,omitempty"`
	CacheCondition   string `form:"cache_condition,omitempty"`
}

// CreateResponseObject creates a new Fastly response object.
func (c *Client) CreateResponseObject(i *CreateResponseObjectInput) (*ResponseObject, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/response_object", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *ResponseObject
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// GetResponseObjectInput is used as input to the GetResponseObject function.
type GetResponseObjectInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the response object to fetch.
	Name string
}

// GetResponseObject gets the response object configuration with the given
// parameters.
func (c *Client) GetResponseObject(i *GetResponseObjectInput) (*ResponseObject, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/response_object/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var b *ResponseObject
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateResponseObjectInput is used as input to the UpdateResponseObject
// function.
type UpdateResponseObjectInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the response object to update.
	Name string

	NewName          string `form:"name,omitempty"`
	Status           uint   `form:"status,omitempty"`
	Response         string `form:"response,omitempty"`
	Content          string `form:"content,omitempty"`
	ContentType      string `form:"content_type,omitempty"`
	RequestCondition string `form:"request_condition,omitempty"`
	CacheCondition   string `form:"cache_condition,omitempty"`
}

// UpdateResponseObject updates a specific response object.
func (c *Client) UpdateResponseObject(i *UpdateResponseObjectInput) (*ResponseObject, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/response_object/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *ResponseObject
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteResponseObjectInput is the input parameter to DeleteResponseObject.
type DeleteResponseObjectInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the response object to delete (required).
	Name string
}

// DeleteResponseObject deletes the given response object version.
func (c *Client) DeleteResponseObject(i *DeleteResponseObjectInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == "" {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/response_object/%s", i.Service, i.Version, i.Name)
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
