// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Metric Cluster API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/metric_cluster

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
)

// MetricQuery object
type MetricQuery struct {
	Query string `json:"query"`
	Type  string `json:"type"`
}

// MetricCluster object
type MetricCluster struct {
	CID                 string              `json:"_cid,omitempty"`
	MatchingMetrics     []string            `json:"_matching_metrics,omitempty"`
	MatchingUUIDMetrics map[string][]string `json:"_matching_uuid_metrics,omitempty"`
	Description         string              `json:"description"`
	Name                string              `json:"name"`
	Queries             []MetricQuery       `json:"queries"`
	Tags                []string            `json:"tags"`
}

const (
	baseMetricClusterPath = "/metric_cluster"
	metricClusterCIDRegex = "^" + baseMetricClusterPath + "/[0-9]+$"
)

// FetchMetricCluster fetch a metric cluster configuration by cid
func (a *API) FetchMetricCluster(cid CIDType, extras string) (*MetricCluster, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid metric cluster CID [none]")
	}

	clusterCID := string(*cid)

	matched, err := regexp.MatchString(metricClusterCIDRegex, clusterCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid metric cluster CID [%s]", clusterCID)
	}

	reqURL := url.URL{
		Path: clusterCID,
	}

	extra := ""
	switch extras {
	case "metrics":
		extra = "_matching_metrics"
	case "uuids":
		extra = "_matching_uuid_metrics"
	}

	if extra != "" {
		q := url.Values{}
		q.Set("extra", extra)
		reqURL.RawQuery = q.Encode()
	}

	resp, err := a.Get(reqURL.String())
	if err != nil {
		return nil, err
	}

	cluster := &MetricCluster{}
	if err := json.Unmarshal(resp, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// FetchMetricClusters fetch metric cluster configurations
func (a *API) FetchMetricClusters(extras string) (*[]MetricCluster, error) {

	reqURL := url.URL{
		Path: baseMetricClusterPath,
	}

	extra := ""
	switch extras {
	case "metrics":
		extra = "_matching_metrics"
	case "uuids":
		extra = "_matching_uuid_metrics"
	}

	if extra != "" {
		q := url.Values{}
		q.Set("extra", extra)
		reqURL.RawQuery = q.Encode()
	}

	resp, err := a.Get(reqURL.String())
	if err != nil {
		return nil, err
	}

	var clusters []MetricCluster
	if err := json.Unmarshal(resp, &clusters); err != nil {
		return nil, err
	}

	return &clusters, nil
}

// UpdateMetricCluster updates a metric cluster
func (a *API) UpdateMetricCluster(config *MetricCluster) (*MetricCluster, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid metric cluster config [nil]")
	}

	clusterCID := string(config.CID)

	if matched, err := regexp.MatchString(metricClusterCIDRegex, clusterCID); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid metric cluster CID [%s]", clusterCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(clusterCID, cfg)
	if err != nil {
		return nil, err
	}

	cluster := &MetricCluster{}
	if err := json.Unmarshal(result, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// CreateMetricCluster create a new metric cluster
func (a *API) CreateMetricCluster(config *MetricCluster) (*MetricCluster, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid metric cluster config [nil]")
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Post(baseMetricClusterPath, cfg)
	if err != nil {
		return nil, err
	}

	cluster := &MetricCluster{}
	if err := json.Unmarshal(result, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// DeleteMetricCluster delete a metric cluster
func (a *API) DeleteMetricCluster(config *MetricCluster) (bool, error) {
	if config == nil {
		return false, fmt.Errorf("Invalid metric cluster config [none]")
	}

	cid := CIDType(&config.CID)
	return a.DeleteMetricClusterByCID(cid)
}

// DeleteMetricClusterByCID delete a metric cluster by cid
func (a *API) DeleteMetricClusterByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid metric cluster CID [none]")
	}

	clusterCID := string(*cid)

	if matched, err := regexp.MatchString(metricClusterCIDRegex, clusterCID); err != nil {
		return false, err
	} else if !matched {
		return false, fmt.Errorf("Invalid metric cluster CID [%s]", clusterCID)
	}

	_, err := a.Delete(clusterCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchMetricClusters returns list of metric clusters matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) SearchMetricClusters(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]MetricCluster, error) {
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
		return a.FetchMetricClusters("")
	}

	reqURL := url.URL{
		Path:     baseMetricClusterPath,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var clusters []MetricCluster
	if err := json.Unmarshal(result, &clusters); err != nil {
		return nil, err
	}

	return &clusters, nil
}
