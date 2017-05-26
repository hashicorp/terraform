// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Metric API support - Fetch, Create*, Update, Delete*, and Search
// See: https://login.circonus.com/resources/api/calls/metric
// *  : create and delete are handled via check_bundle or check_bundle_metrics

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// Metric defines a metric. See https://login.circonus.com/resources/api/calls/metric for more information.
type Metric struct {
	Active         bool     `json:"_active,omitempty"`       // boolean
	CheckActive    bool     `json:"_check_active,omitempty"` // boolean
	CheckBundleCID string   `json:"_check_bundle,omitempty"` // string
	CheckCID       string   `json:"_check,omitempty"`        // string
	CheckTags      []string `json:"_check_tags,omitempty"`   // [] len >= 0
	CheckUUID      string   `json:"_check_uuid,omitempty"`   // string
	CID            string   `json:"_cid,omitempty"`          // string
	Histogram      string   `json:"_histogram,omitempty"`    // string
	Link           *string  `json:"link,omitempty"`          // string or null
	MetricName     string   `json:"_metric_name,omitempty"`  // string
	MetricType     string   `json:"_metric_type,omitempty"`  // string
	Notes          *string  `json:"notes,omitempty"`         // string or null
	Tags           []string `json:"tags,omitempty"`          // [] len >= 0
	Units          *string  `json:"units,omitempty"`         // string or null
}

// FetchMetric retrieves metric with passed cid.
func (a *API) FetchMetric(cid CIDType) (*Metric, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid metric CID [none]")
	}

	metricCID := string(*cid)

	matched, err := regexp.MatchString(config.MetricCIDRegex, metricCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid metric CID [%s]", metricCID)
	}

	result, err := a.Get(metricCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch metric, received JSON: %s", string(result))
	}

	metric := &Metric{}
	if err := json.Unmarshal(result, metric); err != nil {
		return nil, err
	}

	return metric, nil
}

// FetchMetrics retrieves all metrics available to API Token.
func (a *API) FetchMetrics() (*[]Metric, error) {
	result, err := a.Get(config.MetricPrefix)
	if err != nil {
		return nil, err
	}

	var metrics []Metric
	if err := json.Unmarshal(result, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

// UpdateMetric updates passed metric.
func (a *API) UpdateMetric(cfg *Metric) (*Metric, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid metric config [nil]")
	}

	metricCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.MetricCIDRegex, metricCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid metric CID [%s]", metricCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update metric, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(metricCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	metric := &Metric{}
	if err := json.Unmarshal(result, metric); err != nil {
		return nil, err
	}

	return metric, nil
}

// SearchMetrics returns metrics matching the specified search query
// and/or filter. If nil is passed for both parameters all metrics
// will be returned.
func (a *API) SearchMetrics(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Metric, error) {
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
		return a.FetchMetrics()
	}

	reqURL := url.URL{
		Path:     config.MetricPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var metrics []Metric
	if err := json.Unmarshal(result, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}
