// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// CheckDetails is an arbitrary json structure, we would only care about submission_url
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

const baseCheckPath = "/check"

// FetchCheckByID fetch a check configuration by id
func (a *API) FetchCheckByID(id IDType) (*Check, error) {
	cid := CIDType(fmt.Sprintf("%s/%d", baseCheckPath, int(id)))
	return a.FetchCheckByCID(cid)
}

// FetchCheckByCID fetch a check configuration by cid
func (a *API) FetchCheckByCID(cid CIDType) (*Check, error) {
	if matched, err := regexp.MatchString("^"+baseCheckPath+"/[0-9]+$", string(cid)); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid check CID %v", cid)
	}

	result, err := a.Get(string(cid))
	if err != nil {
		return nil, err
	}

	check := new(Check)
	if err := json.Unmarshal(result, check); err != nil {
		return nil, err
	}

	return check, nil
}

// FetchCheckBySubmissionURL fetch a check configuration by submission_url
func (a *API) FetchCheckBySubmissionURL(submissionURL URLType) (*Check, error) {
	if string(submissionURL) == "" {
		return nil, errors.New("[ERROR] Invalid submission URL (blank)")
	}

	u, err := url.Parse(string(submissionURL))
	if err != nil {
		return nil, err
	}

	// valid trap url: scheme://host[:port]/module/httptrap/UUID/secret

	// does it smell like a valid trap url path
	if !strings.Contains(u.Path, "/module/httptrap/") {
		return nil, fmt.Errorf("[ERROR] Invalid submission URL '%s', unrecognized path", submissionURL)
	}

	// extract uuid
	pathParts := strings.Split(strings.Replace(u.Path, "/module/httptrap/", "", 1), "/")
	if len(pathParts) != 2 {
		return nil, fmt.Errorf("[ERROR] Invalid submission URL '%s', UUID not where expected", submissionURL)
	}
	uuid := pathParts[0]

	filter := SearchFilterType(fmt.Sprintf("f__check_uuid=%s", uuid))

	checks, err := a.CheckFilterSearch(filter)
	if err != nil {
		return nil, err
	}

	if len(checks) == 0 {
		return nil, fmt.Errorf("[ERROR] No checks found with UUID %s", uuid)
	}

	numActive := 0
	checkID := -1

	for idx, check := range checks {
		if check.Active {
			numActive++
			checkID = idx
		}
	}

	if numActive > 1 {
		return nil, fmt.Errorf("[ERROR] Multiple checks with same UUID %s", uuid)
	}

	return &checks[checkID], nil

}

// CheckSearch returns a list of checks matching a search query
func (a *API) CheckSearch(searchCriteria SearchQueryType) ([]Check, error) {
	reqURL := url.URL{
		Path: baseCheckPath,
	}

	if searchCriteria != "" {
		q := url.Values{}
		q.Set("search", string(searchCriteria))
		reqURL.RawQuery = q.Encode()
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, err
	}

	var checks []Check
	if err := json.Unmarshal(result, &checks); err != nil {
		return nil, err
	}

	return checks, nil
}

// CheckFilterSearch returns a list of checks matching a filter (filtering allows looking for
// things within sub-elements e.g. details)
func (a *API) CheckFilterSearch(filter SearchFilterType) ([]Check, error) {
	if filter == "" {
		return nil, errors.New("[ERROR] invalid filter supplied (blank)")
	}

	filterURL := fmt.Sprintf("/check?%s", string(filter))

	result, err := a.Get(filterURL)
	if err != nil {
		return nil, err
	}

	var checks []Check
	if err := json.Unmarshal(result, &checks); err != nil {
		return nil, err
	}

	return checks, nil
}
