package librato

import (
	"fmt"
	"net/http"
)

// SpacesService handles communication with the Librato API methods related to
// spaces.
type SpacesService struct {
	client *Client
}

// Space represents a Librato Space.
type Space struct {
	Name *string `json:"name"`
	ID   *uint   `json:"id,omitempty"`
}

func (s Space) String() string {
	return Stringify(s)
}

// SpaceListOptions specifies the optional parameters to the SpaceService.Find
// method.
type SpaceListOptions struct {
	// filter by name
	Name string `url:"name,omitempty"`
}

type listSpacesResponse struct {
	Spaces []Space `json:"spaces"`
}

// List spaces using the provided options.
//
// Librato API docs: http://dev.librato.com/v1/get/spaces
func (s *SpacesService) List(opt *SpaceListOptions) ([]Space, *http.Response, error) {
	u, err := urlWithOptions("spaces", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var spacesResp listSpacesResponse
	resp, err := s.client.Do(req, &spacesResp)
	if err != nil {
		return nil, resp, err
	}

	return spacesResp.Spaces, resp, nil
}

// Get fetches a space based on the provided ID.
//
// Librato API docs: http://dev.librato.com/v1/get/spaces/:id
func (s *SpacesService) Get(id uint) (*Space, *http.Response, error) {
	u, err := urlWithOptions(fmt.Sprintf("spaces/%d", id), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	sp := new(Space)
	resp, err := s.client.Do(req, sp)
	if err != nil {
		return nil, resp, err
	}

	return sp, resp, err
}

// Create a space with a given name.
//
// Librato API docs: http://dev.librato.com/v1/post/spaces
func (s *SpacesService) Create(space *Space) (*Space, *http.Response, error) {
	req, err := s.client.NewRequest("POST", "spaces", space)
	if err != nil {
		return nil, nil, err
	}

	sp := new(Space)
	resp, err := s.client.Do(req, sp)
	if err != nil {
		return nil, resp, err
	}

	return sp, resp, err
}

// Edit a space.
//
// Librato API docs: http://dev.librato.com/v1/put/spaces/:id
func (s *SpacesService) Edit(spaceID uint, space *Space) (*http.Response, error) {
	u := fmt.Sprintf("spaces/%d", spaceID)
	req, err := s.client.NewRequest("PUT", u, space)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// Delete a space.
//
// Librato API docs: http://dev.librato.com/v1/delete/spaces/:id
func (s *SpacesService) Delete(id uint) (*http.Response, error) {
	u := fmt.Sprintf("spaces/%d", id)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
