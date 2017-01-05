// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Check API support - Fetch and Search
// See: https://login.circonus.com/resources/api/calls/check
// Notes: checks do not directly support create, update, and delete - see check bundle.

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
)

// CheckDetails is an arbitrary json structure, contents are undocumented
type CheckDetails struct {
	SubmissionURL string `json:"submission_url"`
}

// Check definition
type Check struct {
	CID            string       `json:"_cid"`
	Active         bool         `json:"_active"`
	BrokerCID      string       `json:"_broker"`
	CheckBundleCID string       `json:"_check_bundle"`
	CheckUUID      string       `json:"_check_uuid"`
	Details        CheckDetails `json:"_details"`
}

const (
	baseCheckPath = "/check"
	checkCIDRegex = "^" + baseCheckPath + "/[0-9]+$"
)

// FetchCheck fetch a check configuration by cid
func (a *API) FetchCheck(cid CIDType) (*Check, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid check CID [none]")
	}

	checkCID := string(*cid)

	matched, err := regexp.MatchString(checkCIDRegex, checkCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid check CID [%s]", checkCID)
	}

	result, err := a.Get(checkCID)
	if err != nil {
		return nil, err
	}

	check := new(Check)
	if err := json.Unmarshal(result, check); err != nil {
		return nil, err
	}

	return check, nil
}

// FetchChecks fetches check configurations
func (a *API) FetchChecks() (*[]Check, error) {
	result, err := a.Get(baseCheckPath)
	if err != nil {
		return nil, err
	}

	var checks []Check
	if err := json.Unmarshal(result, &checks); err != nil {
		return nil, err
	}

	return &checks, nil
}

// SearchChecks returns a list of checks matching a search query
func (a *API) SearchChecks(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Check, error) {
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
		return a.FetchChecks()
	}

	reqURL := url.URL{
		Path:     baseCheckPath,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, err
	}

	var checks []Check
	if err := json.Unmarshal(result, &checks); err != nil {
		return nil, err
	}

	return &checks, nil
}
