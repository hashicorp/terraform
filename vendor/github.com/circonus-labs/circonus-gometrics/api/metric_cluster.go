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

const baseMetricClusterPath = "/metric_cluster"

// FetchMetricClusterByID fetch a metric cluster configuration by id
func (a *API) FetchMetricClusterByID(id IDType, extras string) (*MetricCluster, error) {
	reqURL := url.URL{
		Path: fmt.Sprintf("%s/%d", baseMetricClusterPath, id),
	}
	cid := CIDType(reqURL.String())
	return a.FetchMetricClusterByCID(cid, extras)
}

// FetchMetricClusterByCID fetch a check bundle configuration by id
func (a *API) FetchMetricClusterByCID(cid CIDType, extras string) (*MetricCluster, error) {
	if matched, err := regexp.MatchString("^"+baseMetricClusterPath+"/[0-9]+$", string(cid)); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid metric cluster CID %v", cid)
	}

	reqURL := url.URL{
		Path: string(cid),
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

// MetricClusterSearch returns list of metric clusters matching a search query (or all metric
// clusters if no search query is provided)
//    - a search query not a filter (see: https://login.circonus.com/resources/api#searching)
func (a *API) MetricClusterSearch(searchCriteria SearchQueryType) ([]MetricCluster, error) {
	reqURL := url.URL{
		Path: baseMetricClusterPath,
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

	var clusters []MetricCluster
	if err := json.Unmarshal(resp, &clusters); err != nil {
		return nil, err
	}

	return clusters, nil
}

// CreateMetricCluster create a new metric cluster
func (a *API) CreateMetricCluster(config *MetricCluster) (*MetricCluster, error) {
	reqURL := url.URL{
		Path: baseMetricClusterPath,
	}
	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	resp, err := a.Post(reqURL.String(), cfg)
	if err != nil {
		return nil, err
	}

	cluster := &MetricCluster{}
	if err := json.Unmarshal(resp, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// UpdateMetricCluster updates a metric cluster
func (a *API) UpdateMetricCluster(config *MetricCluster) (*MetricCluster, error) {
	if matched, err := regexp.MatchString("^"+baseMetricClusterPath+"/[0-9]+$", string(config.CID)); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid metric cluster CID %v", config.CID)
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

	cluster := &MetricCluster{}
	if err := json.Unmarshal(resp, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}
