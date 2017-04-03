package contentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// SpacesService model
type SpacesService service

// Space model
type Space struct {
	Sys           *Sys   `json:"sys,omitempty"`
	Name          string `json:"name,omitempty"`
	DefaultLocale string `json:"defaultLocale,omitempty"`
}

// MarshalJSON for custom json marshaling
func (space *Space) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name          string `json:"name,omitempty"`
		DefaultLocale string `json:"defaultLocale,omitempty"`
	}{
		Name:          space.Name,
		DefaultLocale: space.DefaultLocale,
	})
}

// GetVersion returns entity version
func (space *Space) GetVersion() int {
	version := 1
	if space.Sys != nil {
		version = space.Sys.Version
	}

	return version
}

// List creates a spaces collection
func (service *SpacesService) List() *Collection {
	req, _ := service.c.newRequest("GET", "/spaces", nil, nil)

	col := NewCollection(&CollectionOptions{})
	col.c = service.c
	col.req = req

	return col
}

// Get returns a single space entity
func (service *SpacesService) Get(spaceID string) (*Space, error) {
	path := fmt.Sprintf("/spaces/%s", spaceID)
	method := "GET"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return &Space{}, err
	}

	var space Space
	if ok := service.c.do(req, &space); ok != nil {
		return &Space{}, ok
	}

	return &space, nil
}

// Upsert updates or creates a new space
func (service *SpacesService) Upsert(space *Space) error {
	bytesArray, err := json.Marshal(space)
	if err != nil {
		return err
	}

	var path string
	var method string

	if space.Sys != nil && space.Sys.CreatedAt != "" {
		path = fmt.Sprintf("/spaces/%s", space.Sys.ID)
		method = "PUT"
	} else {
		path = "/spaces"
		method = "POST"
	}

	req, err := service.c.newRequest(method, path, nil, bytes.NewReader(bytesArray))
	if err != nil {
		return err
	}

	req.Header.Set("X-Contentful-Version", strconv.Itoa(space.GetVersion()))

	if ok := service.c.do(req, space); ok != nil {
		return ok
	}

	return nil
}

// Delete the given space
func (service *SpacesService) Delete(space *Space) error {
	path := fmt.Sprintf("/spaces/%s", space.Sys.ID)
	method := "DELETE"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return err
	}

	version := strconv.Itoa(space.Sys.Version)
	req.Header.Set("X-Contentful-Version", version)

	if err = service.c.do(req, nil); err != nil {
		return err
	}

	return nil
}
