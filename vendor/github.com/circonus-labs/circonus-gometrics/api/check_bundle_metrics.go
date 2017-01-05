// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// CheckBundleMetrics API support - Fetch, Create*, Update, and Delete**
// See: https://login.circonus.com/resources/api/calls/metrics
// *  : create metrics by adding to array with a status of 'active'
// ** : delete (distable collection of) metrics by changing status from 'active' to 'available'

package api

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// See check_bundle.go for CheckBundleMetric definition

// CheckBundleMetrics defines metrics
type CheckBundleMetrics struct {
	CID     string              `json:"_cid,omitempty"`
	Metrics []CheckBundleMetric `json:"metrics"`
}

const (
	baseCheckBundleMetricsPath = "/check_bundle_metrics"
	metricsCIDRegex            = "^" + baseCheckBundleMetricsPath + "/[0-9]+$"
)

// FetchCheckBundleMetrics retrieves metrics
func (a *API) FetchCheckBundleMetrics(cid CIDType) (*CheckBundleMetrics, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid check bundle metrics CID [none]")
	}

	metricsCID := string(*cid)

	matched, err := regexp.MatchString(metricsCIDRegex, metricsCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid check bundle metrics CID [%s]", metricsCID)
	}

	result, err := a.Get(metricsCID)
	if err != nil {
		return nil, err
	}

	metrics := &CheckBundleMetrics{}
	if err := json.Unmarshal(result, metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

// UpdateCheckBundleMetrics update metrics definition
func (a *API) UpdateCheckBundleMetrics(config *CheckBundleMetrics) (*CheckBundleMetrics, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid check bundle metrics config [nil]")
	}

	metricsCID := string(config.CID)

	matched, err := regexp.MatchString(metricsCIDRegex, metricsCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid check bundle metrics CID [%s]", metricsCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(metricsCID, cfg)
	if err != nil {
		return nil, err
	}

	metrics := &CheckBundleMetrics{}
	if err := json.Unmarshal(result, metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}
