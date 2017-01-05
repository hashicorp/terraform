// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Maintenance window API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/maintenance

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
)

// Maintenance defines a maintenance
type Maintenance struct {
	CID        string      `json:"_cid,omitempty"`
	Item       string      `json:"item,omitempty"`
	Notes      string      `json:"notes,omitempty"`
	Severities interface{} `json:"severities,omitempty"` // CSV string or []string
	Start      uint        `json:"start,omitempty"`
	Stop       uint        `json:"stop,omitempty"`
	Tags       []string    `json:"tags,omitempty"`
	Type       string      `json:"type,omitempty"`
}

const (
	baseMaintenancePath = "/maintenance"
	maintenanceCIDRegex = "^" + baseMaintenancePath + "/[0-9]+$"
)

// FetchMaintenanceWindow retrieves a maintenance window definition
func (a *API) FetchMaintenanceWindow(cid CIDType) (*Maintenance, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid maintenance window CID [none]")
	}

	maintenanceCID := string(*cid)

	matched, err := regexp.MatchString(maintenanceCIDRegex, maintenanceCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid maintenance window CID [%s]", maintenanceCID)
	}

	result, err := a.Get(maintenanceCID)
	if err != nil {
		return nil, err
	}

	window := &Maintenance{}
	if err := json.Unmarshal(result, window); err != nil {
		return nil, err
	}

	return window, nil
}

// FetchMaintenanceWindows retrieves all maintenance windows
func (a *API) FetchMaintenanceWindows() (*[]Maintenance, error) {
	result, err := a.Get(baseMaintenancePath)
	if err != nil {
		return nil, err
	}

	var windows []Maintenance
	if err := json.Unmarshal(result, &windows); err != nil {
		return nil, err
	}

	return &windows, nil
}

// UpdateMaintenanceWindow update maintenance window definition
func (a *API) UpdateMaintenanceWindow(config *Maintenance) (*Maintenance, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid maintenance window config [nil]")
	}

	maintenanceCID := string(config.CID)

	matched, err := regexp.MatchString(maintenanceCIDRegex, maintenanceCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid maintenance window CID [%s]", maintenanceCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(maintenanceCID, cfg)
	if err != nil {
		return nil, err
	}

	window := &Maintenance{}
	if err := json.Unmarshal(result, window); err != nil {
		return nil, err
	}

	return window, nil
}

// CreateMaintenanceWindow create a new maintenance window
func (a *API) CreateMaintenanceWindow(config *Maintenance) (*Maintenance, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid maintenance window config [nil]")
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Post(baseMaintenancePath, cfg)
	if err != nil {
		return nil, err
	}

	window := &Maintenance{}
	if err := json.Unmarshal(result, window); err != nil {
		return nil, err
	}

	return window, nil
}

// DeleteMaintenanceWindow delete a maintenance
func (a *API) DeleteMaintenanceWindow(config *Maintenance) (bool, error) {
	if config == nil {
		return false, fmt.Errorf("Invalid maintenance window config [none]")
	}

	cid := CIDType(&config.CID)
	return a.DeleteMaintenanceWindowByCID(cid)
}

// DeleteMaintenanceWindowByCID delete a maintenance window by cid
func (a *API) DeleteMaintenanceWindowByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid maintenance window CID [none]")
	}

	maintenanceCID := string(*cid)

	matched, err := regexp.MatchString(maintenanceCIDRegex, maintenanceCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid maintenance CID [%s]", maintenanceCID)
	}

	_, err = a.Delete(maintenanceCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchMaintenanceWindows returns list of maintenances matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) SearchMaintenanceWindows(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Maintenance, error) {
	q := url.Values{}

	if searchCriteria != nil && *searchCriteria != "" {
		q.Set("search", string(*searchCriteria))
	}

	if filterCriteria != nil && len(*filterCriteria) > 0 {
		for filter, criteria := range *filterCriteria {
			for _, val := range criteria {
				q.Add(filter, val)
			}
		}
	}

	if q.Encode() == "" {
		return a.FetchMaintenanceWindows()
	}

	reqURL := url.URL{
		Path:     baseMaintenancePath,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var windows []Maintenance
	if err := json.Unmarshal(result, &windows); err != nil {
		return nil, err
	}

	return &windows, nil
}
