// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
)

// CheckBundleConfig configuration specific to check type
type CheckBundleConfig struct {
	AsyncMetrics  bool   `json:"async_metrics"`
	Secret        string `json:"secret"`
	SubmissionURL string `json:"submission_url"`
	ReverseSecret string `json:"reverse:secret_key"`
	HTTPVersion   string `json:"http_version,omitempty"`
	Method        string `json:"method,omitempty"`
	Payload       string `json:"payload,omitempty"`
	Port          string `json:"port,omitempty"`
	ReadLimit     string `json:"read_limit,omitempty"`
	URL           string `json:"url,omitempty"`
}

// CheckBundleMetric individual metric configuration
type CheckBundleMetric struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Units  string   `json:"units"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
}

// CheckBundle definition
type CheckBundle struct {
	CheckUUIDs         []string            `json:"_check_uuids,omitempty"`
	Checks             []string            `json:"_checks,omitempty"`
	CID                string              `json:"_cid,omitempty"`
	Created            int                 `json:"_created,omitempty"`
	LastModified       int                 `json:"_last_modified,omitempty"`
	LastModifedBy      string              `json:"_last_modifed_by,omitempty"`
	ReverseConnectURLs []string            `json:"_reverse_connection_urls"`
	Brokers            []string            `json:"brokers"`
	Config             CheckBundleConfig   `json:"config"`
	DisplayName        string              `json:"display_name"`
	Metrics            []CheckBundleMetric `json:"metrics"`
	MetricLimit        int                 `json:"metric_limit,omitempty"`
	Notes              string              `json:"notes,omitempty"`
	Period             int                 `json:"period,omitempty"`
	Status             string              `json:"status,omitempty"`
	Tags               []string            `json:"tags,omitempty"`
	Target             string              `json:"target"`
	Timeout            int                 `json:"timeout,omitempty"`
	Type               string              `json:"type"`
}

const baseCheckBundlePath = "/check_bundle"

// FetchCheckBundleByID fetch a check bundle configuration by id
func (a *API) FetchCheckBundleByID(id IDType) (*CheckBundle, error) {
	cid := CIDType(fmt.Sprintf("%s/%d", baseCheckBundlePath, id))
	return a.FetchCheckBundleByCID(cid)
}

// FetchCheckBundleByCID fetch a check bundle configuration by id
func (a *API) FetchCheckBundleByCID(cid CIDType) (*CheckBundle, error) {
	if matched, err := regexp.MatchString("^"+baseCheckBundlePath+"/[0-9]+$", string(cid)); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid check bundle CID %v", cid)
	}

	reqURL := url.URL{
		Path: string(cid),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, err
	}

	checkBundle := &CheckBundle{}
	if err := json.Unmarshal(result, checkBundle); err != nil {
		return nil, err
	}

	return checkBundle, nil
}

// CheckBundleSearch returns list of check bundles matching a search query
//    - a search query (see: https://login.circonus.com/resources/api#searching)
func (a *API) CheckBundleSearch(searchCriteria SearchQueryType) ([]CheckBundle, error) {
	reqURL := url.URL{
		Path: baseCheckBundlePath,
	}

	if searchCriteria != "" {
		q := url.Values{}
		q.Set("search", string(searchCriteria))
		reqURL.RawQuery = q.Encode()
	}

	resp, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var results []CheckBundle
	if err := json.Unmarshal(resp, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// CheckBundleFilterSearch returns list of check bundles matching a search query and filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) CheckBundleFilterSearch(searchCriteria SearchQueryType, filterCriteria map[string]string) ([]CheckBundle, error) {
	reqURL := url.URL{
		Path: baseCheckBundlePath,
	}

	if searchCriteria != "" {
		q := url.Values{}
		q.Set("search", string(searchCriteria))
		for field, val := range filterCriteria {
			q.Set(field, val)
		}
		reqURL.RawQuery = q.Encode()
	}

	resp, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var results []CheckBundle
	if err := json.Unmarshal(resp, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// CreateCheckBundle create a new check bundle (check)
func (a *API) CreateCheckBundle(config *CheckBundle) (*CheckBundle, error) {
	reqURL := url.URL{
		Path: baseCheckBundlePath,
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	resp, err := a.Post(reqURL.String(), cfg)
	if err != nil {
		return nil, err
	}

	checkBundle := &CheckBundle{}
	if err := json.Unmarshal(resp, checkBundle); err != nil {
		return nil, err
	}

	return checkBundle, nil
}

// UpdateCheckBundle updates a check bundle configuration
func (a *API) UpdateCheckBundle(config *CheckBundle) (*CheckBundle, error) {
	if matched, err := regexp.MatchString("^"+baseCheckBundlePath+"/[0-9]+$", string(config.CID)); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid check bundle CID %v", config.CID)
	}

	reqURL := url.URL{
		Path: config.CID,
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	resp, err := a.Put(reqURL.String(), cfg)
	if err != nil {
		return nil, err
	}

	checkBundle := &CheckBundle{}
	if err := json.Unmarshal(resp, checkBundle); err != nil {
		return nil, err
	}

	return checkBundle, nil
}
