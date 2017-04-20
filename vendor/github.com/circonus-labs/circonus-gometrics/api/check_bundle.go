// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Check bundle API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/check_bundle

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// CheckBundleMetric individual metric configuration
type CheckBundleMetric struct {
	Name   string   `json:"name"`             // string
	Result *string  `json:"result,omitempty"` // string or null, NOTE not settable - return/information value only
	Status string   `json:"status,omitempty"` // string
	Tags   []string `json:"tags"`             // [] len >= 0
	Type   string   `json:"type"`             // string
	Units  *string  `json:"units,omitempty"`  // string or null

}

// CheckBundleConfig contains the check type specific configuration settings
// as k/v pairs (see https://login.circonus.com/resources/api/calls/check_bundle
// for the specific settings available for each distinct check type)
type CheckBundleConfig map[config.Key]string

// CheckBundle defines a check bundle. See https://login.circonus.com/resources/api/calls/check_bundle for more information.
type CheckBundle struct {
	Brokers            []string            `json:"brokers"`                            // [] len >= 0
	Checks             []string            `json:"_checks,omitempty"`                  // [] len >= 0
	CheckUUIDs         []string            `json:"_check_uuids,omitempty"`             // [] len >= 0
	CID                string              `json:"_cid,omitempty"`                     // string
	Config             CheckBundleConfig   `json:"config"`                             // NOTE contents of config are check type specific, map len >= 0
	Created            uint                `json:"_created,omitempty"`                 // uint
	DisplayName        string              `json:"display_name"`                       // string
	LastModifedBy      string              `json:"_last_modifed_by,omitempty"`         // string
	LastModified       uint                `json:"_last_modified,omitempty"`           // uint
	MetricLimit        int                 `json:"metric_limit,omitempty"`             // int
	Metrics            []CheckBundleMetric `json:"metrics"`                            // [] >= 0
	Notes              *string             `json:"notes,omitempty"`                    // string or null
	Period             uint                `json:"period,omitempty"`                   // uint
	ReverseConnectURLs []string            `json:"_reverse_connection_urls,omitempty"` // [] len >= 0
	Status             string              `json:"status,omitempty"`                   // string
	Tags               []string            `json:"tags,omitempty"`                     // [] len >= 0
	Target             string              `json:"target"`                             // string
	Timeout            float32             `json:"timeout,omitempty"`                  // float32
	Type               string              `json:"type"`                               // string
}

// NewCheckBundle returns new CheckBundle (with defaults, if applicable)
func NewCheckBundle() *CheckBundle {
	return &CheckBundle{
		Config:      make(CheckBundleConfig, config.DefaultConfigOptionsSize),
		MetricLimit: config.DefaultCheckBundleMetricLimit,
		Period:      config.DefaultCheckBundlePeriod,
		Timeout:     config.DefaultCheckBundleTimeout,
		Status:      config.DefaultCheckBundleStatus,
	}
}

// FetchCheckBundle retrieves check bundle with passed cid.
func (a *API) FetchCheckBundle(cid CIDType) (*CheckBundle, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid check bundle CID [none]")
	}

	bundleCID := string(*cid)

	matched, err := regexp.MatchString(config.CheckBundleCIDRegex, bundleCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid check bundle CID [%v]", bundleCID)
	}

	result, err := a.Get(bundleCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch check bundle, received JSON: %s", string(result))
	}

	checkBundle := &CheckBundle{}
	if err := json.Unmarshal(result, checkBundle); err != nil {
		return nil, err
	}

	return checkBundle, nil
}

// FetchCheckBundles retrieves all check bundles available to the API Token.
func (a *API) FetchCheckBundles() (*[]CheckBundle, error) {
	result, err := a.Get(config.CheckBundlePrefix)
	if err != nil {
		return nil, err
	}

	var checkBundles []CheckBundle
	if err := json.Unmarshal(result, &checkBundles); err != nil {
		return nil, err
	}

	return &checkBundles, nil
}

// UpdateCheckBundle updates passed check bundle.
func (a *API) UpdateCheckBundle(cfg *CheckBundle) (*CheckBundle, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid check bundle config [nil]")
	}

	bundleCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.CheckBundleCIDRegex, bundleCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid check bundle CID [%s]", bundleCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update check bundle, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(bundleCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	checkBundle := &CheckBundle{}
	if err := json.Unmarshal(result, checkBundle); err != nil {
		return nil, err
	}

	return checkBundle, nil
}

// CreateCheckBundle creates a new check bundle (check).
func (a *API) CreateCheckBundle(cfg *CheckBundle) (*CheckBundle, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid check bundle config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create check bundle, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.CheckBundlePrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	checkBundle := &CheckBundle{}
	if err := json.Unmarshal(result, checkBundle); err != nil {
		return nil, err
	}

	return checkBundle, nil
}

// DeleteCheckBundle deletes passed check bundle.
func (a *API) DeleteCheckBundle(cfg *CheckBundle) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid check bundle config [nil]")
	}
	return a.DeleteCheckBundleByCID(CIDType(&cfg.CID))
}

// DeleteCheckBundleByCID deletes check bundle with passed cid.
func (a *API) DeleteCheckBundleByCID(cid CIDType) (bool, error) {

	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid check bundle CID [none]")
	}

	bundleCID := string(*cid)

	matched, err := regexp.MatchString(config.CheckBundleCIDRegex, bundleCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid check bundle CID [%v]", bundleCID)
	}

	_, err = a.Delete(bundleCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchCheckBundles returns check bundles matching the specified
// search query and/or filter. If nil is passed for both parameters
// all check bundles will be returned.
func (a *API) SearchCheckBundles(searchCriteria *SearchQueryType, filterCriteria *map[string][]string) (*[]CheckBundle, error) {

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
		return a.FetchCheckBundles()
	}

	reqURL := url.URL{
		Path:     config.CheckBundlePrefix,
		RawQuery: q.Encode(),
	}

	resp, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var results []CheckBundle
	if err := json.Unmarshal(resp, &results); err != nil {
		return nil, err
	}

	return &results, nil
}
