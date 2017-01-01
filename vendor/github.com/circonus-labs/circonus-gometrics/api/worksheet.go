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
)

// WorksheetGraph defines a worksheet cid to be include in the worksheet
type WorksheetGraph struct {
	GraphCID string `json:"graph"`
}

// WorksheetSmartQuery defines a query to include multiple worksheets
type WorksheetSmartQuery struct {
	Name  string   `json:"name"`
	Query string   `json:"query"`
	Order []string `json:"order"`
}

// Worksheet defines a worksheet
type Worksheet struct {
	CID          string                `json:"_cid,omitempty"`
	Description  string                `json:"description"`
	Favorite     bool                  `json:"favorite"`
	Graphs       []WorksheetGraph      `json:"worksheets,omitempty"`
	Notes        string                `json:"notes"`
	SmartQueries []WorksheetSmartQuery `json:"smart_queries,omitempty"`
	Tags         []string              `json:"tags"`
	Title        string                `json:"title"`
}

const (
	baseWorksheetPath = "/worksheet"
	worksheetCIDRegex = "^" + baseWorksheetPath + "/[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{8,12}$"
)

// FetchWorksheet retrieves a worksheet definition
func (a *API) FetchWorksheet(cid CIDType) (*Worksheet, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid worksheet CID [none]")
	}

	worksheetCID := string(*cid)

	matched, err := regexp.MatchString(worksheetCIDRegex, worksheetCID)
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

	worksheet := new(Worksheet)
	if err := json.Unmarshal(result, worksheet); err != nil {
		return nil, err
	}

	return worksheet, nil
}

// FetchWorksheets retrieves all worksheets
func (a *API) FetchWorksheets() (*[]Worksheet, error) {
	result, err := a.Get(baseWorksheetPath)
	if err != nil {
		return nil, err
	}

	var worksheets []Worksheet
	if err := json.Unmarshal(result, &worksheets); err != nil {
		return nil, err
	}

	return &worksheets, nil
}

// UpdateWorksheet update worksheet definition
func (a *API) UpdateWorksheet(config *Worksheet) (*Worksheet, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid worksheet config [nil]")
	}

	worksheetCID := string(config.CID)

	matched, err := regexp.MatchString(worksheetCIDRegex, worksheetCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid worksheet CID [%s]", worksheetCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(worksheetCID, cfg)
	if err != nil {
		return nil, err
	}

	worksheet := &Worksheet{}
	if err := json.Unmarshal(result, worksheet); err != nil {
		return nil, err
	}

	return worksheet, nil
}

// CreateWorksheet create a new worksheet
func (a *API) CreateWorksheet(config *Worksheet) (*Worksheet, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid worksheet config [nil]")
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Post(baseWorksheetPath, cfg)
	if err != nil {
		return nil, err
	}

	worksheet := &Worksheet{}
	if err := json.Unmarshal(result, worksheet); err != nil {
		return nil, err
	}

	return worksheet, nil
}

// DeleteWorksheet delete a worksheet
func (a *API) DeleteWorksheet(config *Worksheet) (bool, error) {
	if config == nil {
		return false, fmt.Errorf("Invalid worksheet config [none]")
	}
	cid := CIDType(&config.CID)
	return a.DeleteWorksheetByCID(cid)
}

// DeleteWorksheetByCID delete a worksheet by cid
func (a *API) DeleteWorksheetByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid worksheet CID [none]")
	}

	worksheetCID := string(*cid)

	matched, err := regexp.MatchString(worksheetCIDRegex, worksheetCID)
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

// SearchWorksheets returns list of worksheets matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
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
		Path:     baseWorksheetPath,
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
