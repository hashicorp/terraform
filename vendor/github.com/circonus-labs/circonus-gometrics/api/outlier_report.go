// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// OutlierReport API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/report

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// OutlierReport defines a outlier report. See https://login.circonus.com/resources/api/calls/report for more information.
type OutlierReport struct {
	CID              string   `json:"_cid,omitempty"`              // string
	Config           string   `json:"config,omitempty"`            // string
	Created          uint     `json:"_created,omitempty"`          // uint
	CreatedBy        string   `json:"_created_by,omitempty"`       // string
	LastModified     uint     `json:"_last_modified,omitempty"`    // uint
	LastModifiedBy   string   `json:"_last_modified_by,omitempty"` // string
	MetricClusterCID string   `json:"metric_cluster,omitempty"`    // st ring
	Tags             []string `json:"tags,omitempty"`              // [] len >= 0
	Title            string   `json:"title,omitempty"`             // string
}

// NewOutlierReport returns a new OutlierReport (with defaults, if applicable)
func NewOutlierReport() *OutlierReport {
	return &OutlierReport{}
}

// FetchOutlierReport retrieves outlier report with passed cid.
func (a *API) FetchOutlierReport(cid CIDType) (*OutlierReport, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid outlier report CID [none]")
	}

	reportCID := string(*cid)

	matched, err := regexp.MatchString(config.OutlierReportCIDRegex, reportCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid outlier report CID [%s]", reportCID)
	}

	result, err := a.Get(reportCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch outlier report, received JSON: %s", string(result))
	}

	report := &OutlierReport{}
	if err := json.Unmarshal(result, report); err != nil {
		return nil, err
	}

	return report, nil
}

// FetchOutlierReports retrieves all outlier reports available to API Token.
func (a *API) FetchOutlierReports() (*[]OutlierReport, error) {
	result, err := a.Get(config.OutlierReportPrefix)
	if err != nil {
		return nil, err
	}

	var reports []OutlierReport
	if err := json.Unmarshal(result, &reports); err != nil {
		return nil, err
	}

	return &reports, nil
}

// UpdateOutlierReport updates passed outlier report.
func (a *API) UpdateOutlierReport(cfg *OutlierReport) (*OutlierReport, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid outlier report config [nil]")
	}

	reportCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.OutlierReportCIDRegex, reportCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid outlier report CID [%s]", reportCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update outlier report, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(reportCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	report := &OutlierReport{}
	if err := json.Unmarshal(result, report); err != nil {
		return nil, err
	}

	return report, nil
}

// CreateOutlierReport creates a new outlier report.
func (a *API) CreateOutlierReport(cfg *OutlierReport) (*OutlierReport, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid outlier report config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create outlier report, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.OutlierReportPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	report := &OutlierReport{}
	if err := json.Unmarshal(result, report); err != nil {
		return nil, err
	}

	return report, nil
}

// DeleteOutlierReport deletes passed outlier report.
func (a *API) DeleteOutlierReport(cfg *OutlierReport) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid outlier report config [nil]")
	}
	return a.DeleteOutlierReportByCID(CIDType(&cfg.CID))
}

// DeleteOutlierReportByCID deletes outlier report with passed cid.
func (a *API) DeleteOutlierReportByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid outlier report CID [none]")
	}

	reportCID := string(*cid)

	matched, err := regexp.MatchString(config.OutlierReportCIDRegex, reportCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid outlier report CID [%s]", reportCID)
	}

	_, err = a.Delete(reportCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchOutlierReports returns outlier report matching the
// specified search query and/or filter. If nil is passed for
// both parameters all outlier report will be returned.
func (a *API) SearchOutlierReports(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]OutlierReport, error) {
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
		return a.FetchOutlierReports()
	}

	reqURL := url.URL{
		Path:     config.OutlierReportPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var reports []OutlierReport
	if err := json.Unmarshal(result, &reports); err != nil {
		return nil, err
	}

	return &reports, nil
}
