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
)

// Metric defines a metric
type Metric struct {
	CID            string   `json:"_cid,omitempty"`
	Active         bool     `json:"_active,omitempty"`
	CheckCID       string   `json:"_check,omitempty"`
	CheckActive    bool     `json:"_check_active,omitempty"`
	CheckBundleCID string   `json:"_check_bundle,omitempty"`
	CheckTags      []string `json:"_check_tags,omitempty"`
	CheckUUID      string   `json:"_check_uuid,omitempty"`
	Histogram      bool     `json:"_histogram,omitempty"`
	MetricName     string   `json:"_metric_name,omitempty"`
	MetricType     string   `json:"_metric_type,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Units          *string  `json:"units,omitempty"` // string or null
	Link           *string  `json:"link,omitempty"`  // string or null
	Notes          *string  `json:"notes,omitempty"` // string or null
}

const (
	baseMetricPath = "/metric"
	metricCIDRegex = "^" + baseMetricPath + "/[0-9]+$"
)

// FetchMetric retrieves a metric definition
func (a *API) FetchMetric(cid CIDType) (*Metric, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid metric CID [none]")
	}

	metricCID := string(*cid)

	matched, err := regexp.MatchString(metricCIDRegex, metricCID)
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

	metric := &Metric{}
	if err := json.Unmarshal(result, metric); err != nil {
		return nil, err
	}

	return metric, nil
}

// FetchMetrics retrieves all metrics
func (a *API) FetchMetrics() (*[]Metric, error) {
	result, err := a.Get(baseMetricPath)
	if err != nil {
		return nil, err
	}

	var metrics []Metric
	if err := json.Unmarshal(result, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

// UpdateMetric update metric definition
func (a *API) UpdateMetric(config *Metric) (*Metric, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid metric config [nil]")
	}

	metricCID := string(config.CID)

	matched, err := regexp.MatchString(metricCIDRegex, metricCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid metric CID [%s]", metricCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(metricCID, cfg)
	if err != nil {
		return nil, err
	}

	metric := &Metric{}
	if err := json.Unmarshal(result, metric); err != nil {
		return nil, err
	}

	return metric, nil
}

// SearchMetrics returns list of metrics matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
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
		Path:     baseMetricPath,
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
