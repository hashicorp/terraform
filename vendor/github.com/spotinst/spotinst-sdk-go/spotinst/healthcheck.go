package spotinst

import (
	"fmt"
	"net/http"
)

// HealthCheckService handles communication with the HealthCheck related
// methods of the Spotinst API.
type HealthCheckService struct {
	client *Client
}

type HealthCheck struct {
	ID         *string            `json:"id,omitempty"`
	Name       *string            `json:"name,omitempty"`
	ResourceID *string            `json:"resourceId,omitempty"`
	Check      *HealthCheckConfig `json:"check,omitempty"`
	*HealthCheckProxy
}

type HealthCheckProxy struct {
	Addr *string `json:"proxyAddress,omitempty"`
	Port *int    `json:"proxyPort,omitempty"`
}

type HealthCheckConfig struct {
	Protocol *string `json:"protocol,omitempty"`
	Endpoint *string `json:"endpoint,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Interval *int    `json:"interval,omitempty"`
	Timeout  *int    `json:"timeout,omitempty"`
	*HealthCheckThreshold
}

type HealthCheckThreshold struct {
	Healthy   *int `json:"healthyThreshold,omitempty"`
	Unhealthy *int `json:"unhealthyThreshold,omitempty"`
}

type HealthCheckResponse struct {
	Response struct {
		Errors []Error        `json:"errors"`
		Items  []*HealthCheck `json:"items"`
	} `json:"response"`
}

type HealthCheckWrapper struct {
	HealthCheck HealthCheck `json:"healthCheck"`
}

// Get an existing HealthCheck.
func (s *HealthCheckService) Get(args ...string) ([]*HealthCheck, *http.Response, error) {
	var id string
	if len(args) > 0 {
		id = args[0]
	}

	path := fmt.Sprintf("healthCheck/%s", id)
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	var retval HealthCheckResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, err
}

// Create a new HealthCheck.
func (s *HealthCheckService) Create(HealthCheck *HealthCheck) ([]*HealthCheck, *http.Response, error) {
	path := "healthCheck"

	req, err := s.client.NewRequest("POST", path, HealthCheckWrapper{HealthCheck: *HealthCheck})
	if err != nil {
		return nil, nil, err
	}

	var retval HealthCheckResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, nil
}

// Update an existing HealthCheck.
func (s *HealthCheckService) Update(HealthCheck *HealthCheck) ([]*HealthCheck, *http.Response, error) {
	id := (*HealthCheck).ID
	(*HealthCheck).ID = nil
	path := fmt.Sprintf("healthCheck/%s", *id)

	req, err := s.client.NewRequest("PUT", path, HealthCheckWrapper{HealthCheck: *HealthCheck})
	if err != nil {
		return nil, nil, err
	}

	var retval HealthCheckResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, nil
}

// Delete an existing HealthCheck.
func (s *HealthCheckService) Delete(HealthCheck *HealthCheck) (*http.Response, error) {
	id := (*HealthCheck).ID
	(*HealthCheck).ID = nil
	path := fmt.Sprintf("healthCheck/%s", *id)

	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// String creates a reasonable string representation.
func (s HealthCheck) String() string {
	return Stringify(s)
}
