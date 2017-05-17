package fastly

import (
	"fmt"
	"sort"
)

// Version represents a distinct configuration version.
type Version struct {
	Number    int    `mapstructure:"number"`
	Comment   string `mapstructure:"comment"`
	ServiceID string `mapstructure:"service_id"`
	Active    bool   `mapstructure:"active"`
	Locked    bool   `mapstructure:"locked"`
	Deployed  bool   `mapstructure:"deployed"`
	Staging   bool   `mapstructure:"staging"`
	Testing   bool   `mapstructure:"testing"`
}

// versionsByNumber is a sortable list of versions. This is used by the version
// `List()` function to sort the API responses.
type versionsByNumber []*Version

// Len, Swap, and Less implement the sortable interface.
func (s versionsByNumber) Len() int      { return len(s) }
func (s versionsByNumber) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s versionsByNumber) Less(i, j int) bool {
	return s[i].Number < s[j].Number
}

// ListVersionsInput is the input to the ListVersions function.
type ListVersionsInput struct {
	// Service is the ID of the service (required).
	Service string
}

// ListVersions returns the full list of all versions of the given service.
func (c *Client) ListVersions(i *ListVersionsInput) ([]*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	path := fmt.Sprintf("/service/%s/version", i.Service)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var e []*Version
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	sort.Sort(versionsByNumber(e))

	return e, nil
}

// LatestVersionInput is the input to the LatestVersion function.
type LatestVersionInput struct {
	// Service is the ID of the service (required).
	Service string
}

// LatestVersion fetches the latest version. If there are no versions, this
// function will return nil (but not an error).
func (c *Client) LatestVersion(i *LatestVersionInput) (*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	list, err := c.ListVersions(&ListVersionsInput{Service: i.Service})
	if err != nil {
		return nil, err
	}
	if len(list) < 1 {
		return nil, nil
	}

	e := list[len(list)-1]
	return e, nil
}

// CreateVersionInput is the input to the CreateVersion function.
type CreateVersionInput struct {
	// Service is the ID of the service (required).
	Service string
}

// CreateVersion constructs a new version. There are no request parameters, but
// you should consult the resulting version number. Note that `CloneVersion` is
// preferred in almost all scenarios, since `Create()` creates a _blank_
// configuration where `Clone()` builds off of an existing configuration.
func (c *Client) CreateVersion(i *CreateVersionInput) (*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	path := fmt.Sprintf("/service/%s/version", i.Service)
	resp, err := c.Post(path, nil)
	if err != nil {
		return nil, err
	}

	var e *Version
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	return e, nil
}

// GetVersionInput is the input to the GetVersion function.
type GetVersionInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the version number to fetch (required).
	Version int
}

// GetVersion fetches a version with the given information.
func (c *Client) GetVersion(i *GetVersionInput) (*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var e *Version
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	return e, nil
}

// UpdateVersionInput is the input to the UpdateVersion function.
type UpdateVersionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int

	Comment string `form:"comment,omitempty"`
}

// UpdateVersion updates the given version
func (c *Client) UpdateVersion(i *UpdateVersionInput) (*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d", i.Service, i.Version)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var e *Version
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	return e, nil
}

// ActivateVersionInput is the input to the ActivateVersion function.
type ActivateVersionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int
}

// ActivateVersion activates the given version.
func (c *Client) ActivateVersion(i *ActivateVersionInput) (*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/activate", i.Service, i.Version)
	resp, err := c.Put(path, nil)
	if err != nil {
		return nil, err
	}

	var e *Version
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	return e, nil
}

// DeactivateVersionInput is the input to the DeactivateVersion function.
type DeactivateVersionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int
}

// DeactivateVersion deactivates the given version.
func (c *Client) DeactivateVersion(i *DeactivateVersionInput) (*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/deactivate", i.Service, i.Version)
	resp, err := c.Put(path, nil)
	if err != nil {
		return nil, err
	}

	var e *Version
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	return e, nil
}

// CloneVersionInput is the input to the CloneVersion function.
type CloneVersionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int
}

// CloneVersion creates a clone of the version with and returns a new
// configuration version with all the same configuration options, but an
// incremented number.
func (c *Client) CloneVersion(i *CloneVersionInput) (*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/clone", i.Service, i.Version)
	resp, err := c.Put(path, nil)
	if err != nil {
		return nil, err
	}

	var e *Version
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	return e, nil
}

// ValidateVersionInput is the input to the ValidateVersion function.
type ValidateVersionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int
}

// ValidateVersion validates if the given version is okay.
func (c *Client) ValidateVersion(i *ValidateVersionInput) (bool, string, error) {
	var msg string

	if i.Service == "" {
		return false, msg, ErrMissingService
	}

	if i.Version == 0 {
		return false, msg, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/validate", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return false, msg, err
	}

	var r *statusResp
	if err := decodeJSON(&r, resp.Body); err != nil {
		return false, msg, err
	}

	msg = r.Msg
	return r.Ok(), msg, nil
}

// LockVersionInput is the input to the LockVersion function.
type LockVersionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version int
}

// LockVersion locks the specified version.
func (c *Client) LockVersion(i *LockVersionInput) (*Version, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == 0 {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%d/lock", i.Service, i.Version)
	resp, err := c.Put(path, nil)
	if err != nil {
		return nil, err
	}

	var e *Version
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	return e, nil
}
