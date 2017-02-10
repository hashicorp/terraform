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

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// CheckDetails contains [undocumented] check type specific information
type CheckDetails map[config.Key]string

// Check defines a check. See https://login.circonus.com/resources/api/calls/check for more information.
type Check struct {
	Active         bool         `json:"_active"`       // bool
	BrokerCID      string       `json:"_broker"`       // string
	CheckBundleCID string       `json:"_check_bundle"` // string
	CheckUUID      string       `json:"_check_uuid"`   // string
	CID            string       `json:"_cid"`          // string
	Details        CheckDetails `json:"_details"`      // NOTE contents of details are check type specific, map len >= 0
}

// FetchCheck retrieves check with passed cid.
func (a *API) FetchCheck(cid CIDType) (*Check, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid check CID [none]")
	}

	checkCID := string(*cid)

	matched, err := regexp.MatchString(config.CheckCIDRegex, checkCID)
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

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch check, received JSON: %s", string(result))
	}

	check := new(Check)
	if err := json.Unmarshal(result, check); err != nil {
		return nil, err
	}

	return check, nil
}

// FetchChecks retrieves all checks available to the API Token.
func (a *API) FetchChecks() (*[]Check, error) {
	result, err := a.Get(config.CheckPrefix)
	if err != nil {
		return nil, err
	}

	var checks []Check
	if err := json.Unmarshal(result, &checks); err != nil {
		return nil, err
	}

	return &checks, nil
}

// SearchChecks returns checks matching the specified search query
// and/or filter. If nil is passed for both parameters all checks
// will be returned.
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
		Path:     config.CheckPrefix,
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
