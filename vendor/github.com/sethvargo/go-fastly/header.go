package fastly

import (
	"fmt"
	"sort"
)

const (
	// HeaderActionSet is a header action that sets or resets a header.
	HeaderActionSet HeaderAction = "set"

	// HeaderActionAppend is a header action that appends to an existing header.
	HeaderActionAppend HeaderAction = "append"

	// HeaderActionDelete is a header action that deletes a header.
	HeaderActionDelete HeaderAction = "delete"

	// HeaderActionRegex is a header action that performs a single regex
	// replacement on a header.
	HeaderActionRegex HeaderAction = "regex"

	// HeaderActionRegexRepeat is a header action that performs a global regex
	// replacement on a header.
	HeaderActionRegexRepeat HeaderAction = "regex_repeat"
)

// HeaderAction is a type of header action.
type HeaderAction string

const (
	// HeaderTypeRequest is a header type that performs on the request before
	// lookups.
	HeaderTypeRequest HeaderType = "request"

	// HeaderTypeFetch is a header type that performs on the request to the origin
	// server.
	HeaderTypeFetch HeaderType = "fetch"

	// HeaderTypeCache is a header type that performs on the response before it's
	// store in the cache.
	HeaderTypeCache HeaderType = "cache"

	// HeaderTypeResponse is a header type that performs on the response before
	// delivering to the client.
	HeaderTypeResponse HeaderType = "response"
)

// HeaderType is a type of header.
type HeaderType string

// Header represents a header response from the Fastly API.
type Header struct {
	ServiceID string `mapstructure:"service_id"`
	Version   string `mapstructure:"version"`

	Name              string       `mapstructure:"name"`
	Action            HeaderAction `mapstructure:"action"`
	IgnoreIfSet       bool         `mapstructure:"ignore_if_set"`
	Type              HeaderType   `mapstructure:"type"`
	Destination       string       `mapstructure:"dst"`
	Source            string       `mapstructure:"src"`
	Regex             string       `mapstructure:"regex"`
	Substitution      string       `mapstructure:"substitution"`
	Priority          uint         `mapstructure:"priority"`
	RequestCondition  string       `mapstructure:"request_condition"`
	CacheCondition    string       `mapstructure:"cache_condition"`
	ResponseCondition string       `mapstructure:"response_condition"`
}

// headersByName is a sortable list of headers.
type headersByName []*Header

// Len, Swap, and Less implement the sortable interface.
func (s headersByName) Len() int      { return len(s) }
func (s headersByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s headersByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListHeadersInput is used as input to the ListHeaders function.
type ListHeadersInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version string
}

// ListHeaders returns the list of headers for the configuration version.
func (c *Client) ListHeaders(i *ListHeadersInput) ([]*Header, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/header", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var bs []*Header
	if err := decodeJSON(&bs, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(headersByName(bs))
	return bs, nil
}

// CreateHeaderInput is used as input to the CreateHeader function.
type CreateHeaderInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	Name              string       `form:"name,omitempty"`
	Action            HeaderAction `form:"action,omitempty"`
	IgnoreIfSet       bool         `form:"ignore_if_set,omitempty"`
	Type              HeaderType   `form:"type,omitempty"`
	Destination       string       `form:"dst,omitempty"`
	Source            string       `form:"src,omitempty"`
	Regex             string       `form:"regex,omitempty"`
	Substitution      string       `form:"substitution,omitempty"`
	Priority          uint         `form:"priority,omitempty"`
	RequestCondition  string       `form:"request_condition,omitempty"`
	CacheCondition    string       `form:"cache_condition,omitempty"`
	ResponseCondition string       `form:"response_condition,omitempty"`
}

// CreateHeader creates a new Fastly header.
func (c *Client) CreateHeader(i *CreateHeaderInput) (*Header, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/header", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *Header
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// GetHeaderInput is used as input to the GetHeader function.
type GetHeaderInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the header to fetch.
	Name string
}

// GetHeader gets the header configuration with the given parameters.
func (c *Client) GetHeader(i *GetHeaderInput) (*Header, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/header/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var b *Header
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateHeaderInput is used as input to the UpdateHeader function.
type UpdateHeaderInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the header to update.
	Name string

	NewName           string       `form:"name,omitempty"`
	Action            HeaderAction `form:"action,omitempty"`
	IgnoreIfSet       bool         `form:"ignore_if_set,omitempty"`
	Type              HeaderType   `form:"type,omitempty"`
	Destination       string       `form:"dst,omitempty"`
	Source            string       `form:"src,omitempty"`
	Regex             string       `form:"regex,omitempty"`
	Substitution      string       `form:"substitution,omitempty"`
	Priority          uint         `form:"priority,omitempty"`
	RequestCondition  string       `form:"request_condition,omitempty"`
	CacheCondition    string       `form:"cache_condition,omitempty"`
	ResponseCondition string       `form:"response_condition,omitempty"`
}

// UpdateHeader updates a specific header.
func (c *Client) UpdateHeader(i *UpdateHeaderInput) (*Header, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/header/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *Header
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteHeaderInput is the input parameter to DeleteHeader.
type DeleteHeaderInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the header to delete (required).
	Name string
}

// DeleteHeader deletes the given header version.
func (c *Client) DeleteHeader(i *DeleteHeaderInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == "" {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/header/%s", i.Service, i.Version, i.Name)
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
