// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// User API support - Fetch, Update, and Search
// See: https://login.circonus.com/resources/api/calls/user
// Note: Create and Delete are not supported directly via the User API
// endpoint. See the Account endpoint for inviting and removing users
// from specific accounts.

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// UserContactInfo defines known contact details
type UserContactInfo struct {
	SMS  string `json:"sms,omitempty"`
	XMPP string `json:"xmpp,omitempty"`
}

// User definition
type User struct {
	CID         string          `json:"_cid,omitempty"`
	ContactInfo UserContactInfo `json:"contact_info,omitempty"`
	Email       string          `json:"email"`
	Firstname   string          `json:"firstname"`
	Lastname    string          `json:"lastname"`
}

// FetchUser retrieves a user definition
func (a *API) FetchUser(cid CIDType) (*User, error) {
	var userCID string

	if cid == nil || *cid == "" {
		userCID = config.UserPrefix + "/current"
	} else {
		userCID = string(*cid)
	}

	matched, err := regexp.MatchString(config.UserCIDRegex, userCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid user CID [%s]", userCID)
	}

	result, err := a.Get(userCID)
	if err != nil {
		return nil, err
	}

	user := new(User)
	if err := json.Unmarshal(result, user); err != nil {
		return nil, err
	}

	return user, nil
}

// FetchUsers retrieves users for current account
func (a *API) FetchUsers() (*[]User, error) {
	result, err := a.Get(config.UserPrefix)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(result, &users); err != nil {
		return nil, err
	}

	return &users, nil
}

// UpdateUser update user information
func (a *API) UpdateUser(cfg *User) (*User, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid user config [nil]")
	}

	userCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.UserCIDRegex, userCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid user CID [%s]", userCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(userCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	user := &User{}
	if err := json.Unmarshal(result, user); err != nil {
		return nil, err
	}

	return user, nil
}

// SearchUsers returns list of users matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) SearchUsers(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]User, error) {
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
		return a.FetchUsers()
	}

	reqURL := url.URL{
		Path:     config.UserPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var users []User
	if err := json.Unmarshal(result, &users); err != nil {
		return nil, err
	}

	return &users, nil
}
