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

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// Maintenance defines a maintenance window. See https://login.circonus.com/resources/api/calls/maintenance for more information.
type Maintenance struct {
	CID        string      `json:"_cid,omitempty"`       // string
	Item       string      `json:"item,omitempty"`       // string
	Notes      string      `json:"notes,omitempty"`      // string
	Severities interface{} `json:"severities,omitempty"` // []string NOTE can be set with CSV string or []string
	Start      uint        `json:"start,omitempty"`      // uint
	Stop       uint        `json:"stop,omitempty"`       // uint
	Tags       []string    `json:"tags,omitempty"`       // [] len >= 0
	Type       string      `json:"type,omitempty"`       // string
}

// NewMaintenanceWindow returns a new Maintenance window (with defaults, if applicable)
func NewMaintenanceWindow() *Maintenance {
	return &Maintenance{}
}

// FetchMaintenanceWindow retrieves maintenance [window] with passed cid.
func (a *API) FetchMaintenanceWindow(cid CIDType) (*Maintenance, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid maintenance window CID [none]")
	}

	maintenanceCID := string(*cid)

	matched, err := regexp.MatchString(config.MaintenanceCIDRegex, maintenanceCID)
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

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch maintenance window, received JSON: %s", string(result))
	}

	window := &Maintenance{}
	if err := json.Unmarshal(result, window); err != nil {
		return nil, err
	}

	return window, nil
}

// FetchMaintenanceWindows retrieves all maintenance [windows] available to API Token.
func (a *API) FetchMaintenanceWindows() (*[]Maintenance, error) {
	result, err := a.Get(config.MaintenancePrefix)
	if err != nil {
		return nil, err
	}

	var windows []Maintenance
	if err := json.Unmarshal(result, &windows); err != nil {
		return nil, err
	}

	return &windows, nil
}

// UpdateMaintenanceWindow updates passed maintenance [window].
func (a *API) UpdateMaintenanceWindow(cfg *Maintenance) (*Maintenance, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid maintenance window config [nil]")
	}

	maintenanceCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.MaintenanceCIDRegex, maintenanceCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid maintenance window CID [%s]", maintenanceCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update maintenance window, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(maintenanceCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	window := &Maintenance{}
	if err := json.Unmarshal(result, window); err != nil {
		return nil, err
	}

	return window, nil
}

// CreateMaintenanceWindow creates a new maintenance [window].
func (a *API) CreateMaintenanceWindow(cfg *Maintenance) (*Maintenance, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid maintenance window config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create maintenance window, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.MaintenancePrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	window := &Maintenance{}
	if err := json.Unmarshal(result, window); err != nil {
		return nil, err
	}

	return window, nil
}

// DeleteMaintenanceWindow deletes passed maintenance [window].
func (a *API) DeleteMaintenanceWindow(cfg *Maintenance) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid maintenance window config [nil]")
	}
	return a.DeleteMaintenanceWindowByCID(CIDType(&cfg.CID))
}

// DeleteMaintenanceWindowByCID deletes maintenance [window] with passed cid.
func (a *API) DeleteMaintenanceWindowByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid maintenance window CID [none]")
	}

	maintenanceCID := string(*cid)

	matched, err := regexp.MatchString(config.MaintenanceCIDRegex, maintenanceCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid maintenance window CID [%s]", maintenanceCID)
	}

	_, err = a.Delete(maintenanceCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchMaintenanceWindows returns maintenance [windows] matching
// the specified search query and/or filter. If nil is passed for
// both parameters all maintenance [windows] will be returned.
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
		Path:     config.MaintenancePrefix,
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
