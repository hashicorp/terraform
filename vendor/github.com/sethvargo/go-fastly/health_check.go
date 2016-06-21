package fastly

import (
	"fmt"
	"sort"
)

// HealthCheck represents a health check response from the Fastly API.
type HealthCheck struct {
	ServiceID string `mapstructure:"service_id"`
	Version   string `mapstructure:"version"`

	Name             string `mapstructure:"name"`
	Method           string `mapstructure:"method"`
	Host             string `mapstructure:"host"`
	Path             string `mapstructure:"path"`
	HTTPVersion      string `mapstructure:"http_version"`
	Timeout          uint   `mapstructure:"timeout"`
	CheckInterval    uint   `mapstructure:"check_interval"`
	ExpectedResponse uint   `mapstructure:"expected_response"`
	Window           uint   `mapstructure:"window"`
	Threshold        uint   `mapstructure:"threshold"`
	Initial          uint   `mapstructure:"initial"`
}

// healthChecksByName is a sortable list of health checks.
type healthChecksByName []*HealthCheck

// Len, Swap, and Less implement the sortable interface.
func (s healthChecksByName) Len() int      { return len(s) }
func (s healthChecksByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s healthChecksByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListHealthChecksInput is used as input to the ListHealthChecks function.
type ListHealthChecksInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version string
}

// ListHealthChecks returns the list of health checks for the configuration
// version.
func (c *Client) ListHealthChecks(i *ListHealthChecksInput) ([]*HealthCheck, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/healthcheck", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var hcs []*HealthCheck
	if err := decodeJSON(&hcs, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(healthChecksByName(hcs))
	return hcs, nil
}

// CreateHealthCheckInput is used as input to the CreateHealthCheck function.
type CreateHealthCheckInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	Name             string `form:"name,omitempty"`
	Method           string `form:"method,omitempty"`
	Host             string `form:"host,omitempty"`
	Path             string `form:"path,omitempty"`
	HTTPVersion      string `form:"http_version,omitempty"`
	Timeout          uint   `form:"timeout,omitempty"`
	CheckInterval    uint   `form:"check_interval,omitempty"`
	ExpectedResponse uint   `form:"expected_response,omitempty"`
	Window           uint   `form:"window,omitempty"`
	Threshold        uint   `form:"threshold,omitempty"`
	Initial          uint   `form:"initial,omitempty"`
}

// CreateHealthCheck creates a new Fastly health check.
func (c *Client) CreateHealthCheck(i *CreateHealthCheckInput) (*HealthCheck, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/healthcheck", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var h *HealthCheck
	if err := decodeJSON(&h, resp.Body); err != nil {
		return nil, err
	}
	return h, nil
}

// GetHealthCheckInput is used as input to the GetHealthCheck function.
type GetHealthCheckInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the health check to fetch.
	Name string
}

// GetHealthCheck gets the health check configuration with the given parameters.
func (c *Client) GetHealthCheck(i *GetHealthCheckInput) (*HealthCheck, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/healthcheck/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var h *HealthCheck
	if err := decodeJSON(&h, resp.Body); err != nil {
		return nil, err
	}
	return h, nil
}

// UpdateHealthCheckInput is used as input to the UpdateHealthCheck function.
type UpdateHealthCheckInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the health check to update.
	Name string

	NewName          string `form:"name,omitempty"`
	Method           string `form:"method,omitempty"`
	Host             string `form:"host,omitempty"`
	Path             string `form:"path,omitempty"`
	HTTPVersion      string `form:"http_version,omitempty"`
	Timeout          uint   `form:"timeout,omitempty"`
	CheckInterval    uint   `form:"check_interval,omitempty"`
	ExpectedResponse uint   `form:"expected_response,omitempty"`
	Window           uint   `form:"window,omitempty"`
	Threshold        uint   `form:"threshold,omitempty"`
	Initial          uint   `form:"initial,omitempty"`
}

// UpdateHealthCheck updates a specific health check.
func (c *Client) UpdateHealthCheck(i *UpdateHealthCheckInput) (*HealthCheck, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/healthcheck/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var h *HealthCheck
	if err := decodeJSON(&h, resp.Body); err != nil {
		return nil, err
	}
	return h, nil
}

// DeleteHealthCheckInput is the input parameter to DeleteHealthCheck.
type DeleteHealthCheckInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the health check to delete (required).
	Name string
}

// DeleteHealthCheck deletes the given health check.
func (c *Client) DeleteHealthCheck(i *DeleteHealthCheckInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == "" {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/healthcheck/%s", i.Service, i.Version, i.Name)
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
