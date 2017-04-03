package contentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// LocalesService service
type LocalesService service

// Locale model
type Locale struct {
	Sys *Sys `json:"sys,omitempty"`

	// Locale name
	Name string `json:"name,omitempty"`

	// Language code
	Code string `json:"code,omitempty"`

	// If no content is provided for the locale, the Delivery API will return content in a locale specified below:
	FallbackCode string `json:"fallbackCode,omitempty"`

	// Make the locale as default locale for your account
	Default bool `json:"default,omitempty"`

	// Entries with required fields can still be published if locale is empty.
	Optional bool `json:"optional,omitempty"`

	// Includes locale in the Delivery API response.
	CDA bool `json:"contentDeliveryApi"`

	// Displays locale to editors and enables it in Management API.
	CMA bool `json:"contentManagementApi"`
}

// GetVersion returns entity version
func (locale *Locale) GetVersion() int {
	version := 1
	if locale.Sys != nil {
		version = locale.Sys.Version
	}

	return version
}

// List returns a locales collection
func (service *LocalesService) List(spaceID string) *Collection {
	path := fmt.Sprintf("/spaces/%s/locales", spaceID)
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

// Get returns a single locale entity
func (service *LocalesService) Get(spaceID, localeID string) (*Locale, error) {
	path := fmt.Sprintf("/spaces/%s/locales/%s", spaceID, localeID)
	method := "GET"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var locale Locale
	if err := service.c.do(req, &locale); err != nil {
		return nil, err
	}

	return &locale, nil
}

// Delete the locale
func (service *LocalesService) Delete(spaceID string, locale *Locale) error {
	path := fmt.Sprintf("/spaces/%s/locales/%s", spaceID, locale.Sys.ID)
	method := "DELETE"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return err
	}

	version := strconv.Itoa(locale.Sys.Version)
	req.Header.Set("X-Contentful-Version", version)

	if err = service.c.do(req, nil); err != nil {
		return err
	}

	return nil
}

// Upsert updates or creates a new locale entity
func (service *LocalesService) Upsert(spaceID string, locale *Locale) error {
	bytesArray, err := json.Marshal(locale)
	if err != nil {
		return err
	}

	var path string
	var method string

	if locale.Sys != nil && locale.Sys.CreatedAt != "" {
		path = fmt.Sprintf("/spaces/%s/locales/%s", spaceID, locale.Sys.ID)
		method = "PUT"
	} else {
		path = fmt.Sprintf("/spaces/%s/locales", spaceID)
		method = "POST"
	}

	req, err := service.c.newRequest(method, path, nil, bytes.NewReader(bytesArray))
	if err != nil {
		return err
	}

	req.Header.Set("X-Contentful-Version", strconv.Itoa(locale.GetVersion()))

	if ok := service.c.do(req, locale); ok != nil {
		return ok
	}

	return nil
}
