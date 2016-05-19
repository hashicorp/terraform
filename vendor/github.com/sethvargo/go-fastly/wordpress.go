package fastly

import (
	"fmt"
	"sort"
)

// Wordpress represents a wordpress response from the Fastly API.
type Wordpress struct {
	ServiceID string `mapstructure:"service_id"`
	Version   string `mapstructure:"version"`

	Name    string `mapstructure:"name"`
	Path    string `mapstructure:"path"`
	Comment string `mapstructure:"comment"`
}

// wordpressesByName is a sortable list of wordpresses.
type wordpressesByName []*Wordpress

// Len, Swap, and Less implement the sortable interface.
func (s wordpressesByName) Len() int      { return len(s) }
func (s wordpressesByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s wordpressesByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListWordpressesInput is used as input to the ListWordpresses function.
type ListWordpressesInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string
}

// ListWordpresses returns the list of wordpresses for the configuration version.
func (c *Client) ListWordpresses(i *ListWordpressesInput) ([]*Wordpress, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/wordpress", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var bs []*Wordpress
	if err := decodeJSON(&bs, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(wordpressesByName(bs))
	return bs, nil
}

// CreateWordpressInput is used as input to the CreateWordpress function.
type CreateWordpressInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	Name    string `form:"name,omitempty"`
	Path    string `form:"path,omitempty"`
	Comment string `form:"comment,omitempty"`
}

// CreateWordpress creates a new Fastly wordpress.
func (c *Client) CreateWordpress(i *CreateWordpressInput) (*Wordpress, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/wordpress", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *Wordpress
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// GetWordpressInput is used as input to the GetWordpress function.
type GetWordpressInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the wordpress to fetch.
	Name string
}

// GetWordpress gets the wordpress configuration with the given parameters.
func (c *Client) GetWordpress(i *GetWordpressInput) (*Wordpress, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/wordpress/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var b *Wordpress
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateWordpressInput is used as input to the UpdateWordpress function.
type UpdateWordpressInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the wordpress to update.
	Name string

	NewName string `form:"name,omitempty"`
	Path    string `form:"path,omitempty"`
	Comment string `form:"comment,omitempty"`
}

// UpdateWordpress updates a specific wordpress.
func (c *Client) UpdateWordpress(i *UpdateWordpressInput) (*Wordpress, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/wordpress/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *Wordpress
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteWordpressInput is the input parameter to DeleteWordpress.
type DeleteWordpressInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the wordpress to delete (required).
	Name string
}

// DeleteWordpress deletes the given wordpress version.
func (c *Client) DeleteWordpress(i *DeleteWordpressInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == "" {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/wordpress/%s", i.Service, i.Version, i.Name)
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
