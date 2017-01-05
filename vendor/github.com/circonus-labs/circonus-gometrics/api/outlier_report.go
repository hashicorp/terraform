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
)

// OutlierReport defines a outlier report
type OutlierReport struct {
	CID              string   `json:"_cid,omitempty"`
	Created          uint     `json:"_created,omitempty"`
	CreatedBy        string   `json:"_created_by,omitempty"`
	LastModified     uint     `json:"_last_modified,omitempty"`
	LastModifiedBy   string   `json:"_last_modified_by,omitempty"`
	Config           string   `json:"config,omitempty"`
	MetricClusterCID string   `json:"metric_cluster,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Title            string   `json:"title,omitempty"`
}

const (
	baseOutlierReportPath = "/outlier_report"
	reportCIDRegex        = "^" + baseOutlierReportPath + "/[0-9]+$"
)

// FetchOutlierReport retrieves a outlier report definition
func (a *API) FetchOutlierReport(cid CIDType) (*OutlierReport, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid outlier report CID [none]")
	}

	reportCID := string(*cid)

	matched, err := regexp.MatchString(reportCIDRegex, reportCID)
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

	report := &OutlierReport{}
	if err := json.Unmarshal(result, report); err != nil {
		return nil, err
	}

	return report, nil
}

// FetchOutlierReports retrieves all outlier reports
func (a *API) FetchOutlierReports() (*[]OutlierReport, error) {
	result, err := a.Get(baseOutlierReportPath)
	if err != nil {
		return nil, err
	}

	var reports []OutlierReport
	if err := json.Unmarshal(result, &reports); err != nil {
		return nil, err
	}

	return &reports, nil
}

// UpdateOutlierReport update outlier report definition
func (a *API) UpdateOutlierReport(config *OutlierReport) (*OutlierReport, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid outlier report config [nil]")
	}

	reportCID := string(config.CID)

	matched, err := regexp.MatchString(reportCIDRegex, reportCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid outlier report CID [%s]", reportCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(reportCID, cfg)
	if err != nil {
		return nil, err
	}

	report := &OutlierReport{}
	if err := json.Unmarshal(result, report); err != nil {
		return nil, err
	}

	return report, nil
}

// CreateOutlierReport create a new outlier report
func (a *API) CreateOutlierReport(config *OutlierReport) (*OutlierReport, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid outlier report config [nil]")
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Post(baseOutlierReportPath, cfg)
	if err != nil {
		return nil, err
	}

	report := &OutlierReport{}
	if err := json.Unmarshal(result, report); err != nil {
		return nil, err
	}

	return report, nil
}

// DeleteOutlierReport delete a report
func (a *API) DeleteOutlierReport(config *OutlierReport) (bool, error) {
	if config == nil {
		return false, fmt.Errorf("Invalid report config [none]")
	}

	cid := CIDType(&config.CID)
	return a.DeleteOutlierReportByCID(cid)
}

// DeleteOutlierReportByCID delete a outlier report by cid
func (a *API) DeleteOutlierReportByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid outlier report CID [none]")
	}

	reportCID := string(*cid)

	matched, err := regexp.MatchString(reportCIDRegex, reportCID)
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

// SearchOutlierReports returns list of outlier reports matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
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
		Path:     baseOutlierReportPath,
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
