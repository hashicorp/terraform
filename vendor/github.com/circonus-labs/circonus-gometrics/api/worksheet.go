// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Worksheet API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/worksheet

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// WorksheetGraph defines a worksheet cid to be include in the worksheet
type WorksheetGraph struct {
	GraphCID string `json:"graph"` // string
}

// WorksheetSmartQuery defines a query to include multiple worksheets
type WorksheetSmartQuery struct {
	Name  string   `json:"name"`
	Order []string `json:"order"`
	Query string   `json:"query"`
}

// Worksheet defines a worksheet. See https://login.circonus.com/resources/api/calls/worksheet for more information.
type Worksheet struct {
	CID          string                `json:"_cid,omitempty"`          // string
	Description  *string               `json:"description"`             // string or null
	Favorite     bool                  `json:"favorite"`                // boolean
	Graphs       []WorksheetGraph      `json:"worksheets,omitempty"`    // [] len >= 0
	Notes        *string               `json:"notes"`                   // string or null
	SmartQueries []WorksheetSmartQuery `json:"smart_queries,omitempty"` // [] len >= 0
	Tags         []string              `json:"tags"`                    // [] len >= 0
	Title        string                `json:"title"`                   // string
}

// NewWorksheet returns a new Worksheet (with defaults, if applicable)
func NewWorksheet() *Worksheet {
	return &Worksheet{}
}

// FetchWorksheet retrieves worksheet with passed cid.
func (a *API) FetchWorksheet(cid CIDType) (*Worksheet, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid worksheet CID [none]")
	}

	worksheetCID := string(*cid)

	matched, err := regexp.MatchString(config.WorksheetCIDRegex, worksheetCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid worksheet CID [%s]", worksheetCID)
	}

	result, err := a.Get(string(*cid))
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch worksheet, received JSON: %s", string(result))
	}

	worksheet := new(Worksheet)
	if err := json.Unmarshal(result, worksheet); err != nil {
		return nil, err
	}

	return worksheet, nil
}

// FetchWorksheets retrieves all worksheets available to API Token.
func (a *API) FetchWorksheets() (*[]Worksheet, error) {
	result, err := a.Get(config.WorksheetPrefix)
	if err != nil {
		return nil, err
	}

	var worksheets []Worksheet
	if err := json.Unmarshal(result, &worksheets); err != nil {
		return nil, err
	}

	return &worksheets, nil
}

// UpdateWorksheet updates passed worksheet.
func (a *API) UpdateWorksheet(cfg *Worksheet) (*Worksheet, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid worksheet config [nil]")
	}

	worksheetCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.WorksheetCIDRegex, worksheetCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid worksheet CID [%s]", worksheetCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update worksheet, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(worksheetCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	worksheet := &Worksheet{}
	if err := json.Unmarshal(result, worksheet); err != nil {
		return nil, err
	}

	return worksheet, nil
}

// CreateWorksheet creates a new worksheet.
func (a *API) CreateWorksheet(cfg *Worksheet) (*Worksheet, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid worksheet config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create annotation, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.WorksheetPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	worksheet := &Worksheet{}
	if err := json.Unmarshal(result, worksheet); err != nil {
		return nil, err
	}

	return worksheet, nil
}

// DeleteWorksheet deletes passed worksheet.
func (a *API) DeleteWorksheet(cfg *Worksheet) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid worksheet config [nil]")
	}
	return a.DeleteWorksheetByCID(CIDType(&cfg.CID))
}

// DeleteWorksheetByCID deletes worksheet with passed cid.
func (a *API) DeleteWorksheetByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid worksheet CID [none]")
	}

	worksheetCID := string(*cid)

	matched, err := regexp.MatchString(config.WorksheetCIDRegex, worksheetCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid worksheet CID [%s]", worksheetCID)
	}

	_, err = a.Delete(worksheetCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchWorksheets returns worksheets matching the specified search
// query and/or filter. If nil is passed for both parameters all
// worksheets will be returned.
func (a *API) SearchWorksheets(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Worksheet, error) {
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
		return a.FetchWorksheets()
	}

	reqURL := url.URL{
		Path:     config.WorksheetPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var worksheets []Worksheet
	if err := json.Unmarshal(result, &worksheets); err != nil {
		return nil, err
	}

	return &worksheets, nil
}
