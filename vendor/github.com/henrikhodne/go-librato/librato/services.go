package librato

import (
	"fmt"
	"net/http"
)

// ServicesService handles communication with the Librato API methods related to
// notification services.
type ServicesService struct {
	client *Client
}

// Service represents a Librato Service.
type Service struct {
	ID    *uint   `json:"id,omitempty"`
	Type  *string `json:"type,omitempty"`
	Title *string `json:"title,omitempty"`
	// This is an interface{} because it's a hash of settings
	// specific to each service.
	Settings map[string]string `json:"settings,omitempty"`
}

func (a Service) String() string {
	return Stringify(a)
}

// Get a service by ID
//
// Librato API docs: https://www.librato.com/docs/api/#retrieve-specific-service
func (s *ServicesService) Get(id uint) (*Service, *http.Response, error) {
	urlStr := fmt.Sprintf("services/%d", id)

	req, err := s.client.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, nil, err
	}

	service := new(Service)
	resp, err := s.client.Do(req, service)
	if err != nil {
		return nil, resp, err
	}

	return service, resp, err
}

// Create a service
//
// Librato API docs: https://www.librato.com/docs/api/#create-a-service
func (s *ServicesService) Create(service *Service) (*Service, *http.Response, error) {
	req, err := s.client.NewRequest("POST", "services", service)
	if err != nil {
		return nil, nil, err
	}

	sv := new(Service)
	resp, err := s.client.Do(req, sv)
	if err != nil {
		return nil, resp, err
	}

	return sv, resp, err
}

// Update a service.
//
// Librato API docs: https://www.librato.com/docs/api/#update-a-service
func (s *ServicesService) Update(serviceID uint, service *Service) (*http.Response, error) {
	u := fmt.Sprintf("services/%d", serviceID)
	req, err := s.client.NewRequest("PUT", u, service)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// Delete a service
//
// Librato API docs: https://www.librato.com/docs/api/#delete-a-service
func (s *ServicesService) Delete(id uint) (*http.Response, error) {
	u := fmt.Sprintf("services/%d", id)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
