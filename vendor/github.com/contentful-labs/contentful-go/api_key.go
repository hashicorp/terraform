package contentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// APIKeyService service
type APIKeyService service

// APIKey model
type APIKey struct {
	Sys           *Sys            `json:"sys,omitempty"`
	Name          string          `json:"name,omitempty"`
	Description   string          `json:"description,omitempty"`
	AccessToken   string          `json:"accessToken,omitempty"`
	Policies      []*APIKeyPolicy `json:"policies,omitempty"`
	PreviewAPIKey *PreviewAPIKey  `json:"preview_api_key,omitempty"`
}

// APIKeyPolicy model
type APIKeyPolicy struct {
	Effect  string `json:"effect,omitempty"`
	Actions string `json:"actions,omitempty"`
}

// PreviewAPIKey model
type PreviewAPIKey struct {
	Sys *Sys
}

// MarshalJSON for custom json marshaling
func (apiKey *APIKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}{
		Name:        apiKey.Name,
		Description: apiKey.Description,
	})
}

// GetVersion returns entity version
func (apiKey *APIKey) GetVersion() int {
	version := 1
	if apiKey.Sys != nil {
		version = apiKey.Sys.Version
	}

	return version
}

// List returns all api keys collection
func (service *APIKeyService) List(spaceID string) *Collection {
	path := fmt.Sprintf("/spaces/%s/api_keys", spaceID)
	method := "GET"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return &Collection{}
	}

	col := NewCollection(&CollectionOptions{})
	col.c = service.c
	col.req = req

	return col
}

// Get returns a single api key entity
func (service *APIKeyService) Get(spaceID, apiKeyID string) (*APIKey, error) {
	path := fmt.Sprintf("/spaces/%s/api_keys/%s", spaceID, apiKeyID)
	method := "GET"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var apiKey APIKey
	if err := service.c.do(req, &apiKey); err != nil {
		return nil, err
	}

	return &apiKey, nil
}

// Upsert updates or creates a new api key entity
func (service *APIKeyService) Upsert(spaceID string, apiKey *APIKey) error {
	bytesArray, err := json.Marshal(apiKey)
	if err != nil {
		return err
	}

	var path string
	var method string

	if apiKey.Sys != nil && apiKey.Sys.CreatedAt != "" {
		path = fmt.Sprintf("/spaces/%s/api_keys/%s", spaceID, apiKey.Sys.ID)
		method = "PUT"
	} else {
		path = fmt.Sprintf("/spaces/%s/api_keys", spaceID)
		method = "POST"
	}

	req, err := service.c.newRequest(method, path, nil, bytes.NewReader(bytesArray))
	if err != nil {
		return err
	}

	req.Header.Set("X-Contentful-Version", strconv.Itoa(apiKey.GetVersion()))

	if ok := service.c.do(req, apiKey); ok != nil {
		return ok
	}

	return nil
}

// Delete deletes a sinlge api key entity
func (service *APIKeyService) Delete(spaceID string, apiKey *APIKey) error {
	path := fmt.Sprintf("/spaces/%s/api_keys/%s", spaceID, apiKey.Sys.ID)
	method := "DELETE"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return err
	}

	version := strconv.Itoa(apiKey.Sys.Version)
	req.Header.Set("X-Contentful-Version", version)

	if err = service.c.do(req, nil); err != nil {
		return err
	}

	return nil
}
