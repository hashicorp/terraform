// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Graph API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/graph

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
)

// GraphAccessKey defines an access key for a graph
type GraphAccessKey struct {
	Active         bool   `json:"active,omitempty"`
	Height         int    `json:"height,omitempty"`
	Key            string `json:"key,omitempty"`
	Legend         bool   `json:"legend,omitempty"`
	LockDate       bool   `json:"lock_date,omitempty"`
	LockMode       string `json:"lock_mode,omitempty"`
	LockRangeEnd   int    `json:"lock_range_end,omitempty"`
	LockRangeStart int    `json:"lock_range_start,omitempty"`
	LockShowTimes  bool   `json:"lock_show_times,omitempty"`
	LockZoom       string `json:"lock_zoom,omitempty"`
	Nickname       string `json:"nickname,omitempty"`
	Title          bool   `json:"title,omitempty"`
	Width          int    `json:"width,omitempty"`
	XLabels        bool   `json:"x_labels,omitempty"`
	YLabels        bool   `json:"y_labels,omitempty"`
}

// GraphComposite defines a composite
type GraphComposite struct {
	Axis          string `json:"axis,omitempty"`
	Color         string `json:"color,omitempty"`
	DataFormula   string `json:"data_formula,omitempty"`
	Hidden        bool   `json:"hidden,omitempty"`
	LegendFormula string `json:"legend_formula,omitempty"`
	Name          string `json:"name,omitempty"`
	Stack         int    `json:"stack,omitempty"`
}

// GraphDatapoint defines a datapoint
type GraphDatapoint struct {
	Alpha         string      `json:"alpha,omitempty"`
	Axis          string      `json:"axis,omitempty"`
	CAQL          string      `json:"caql,omitempty"`
	CheckID       int         `json:"check_id,omitempty"`
	Color         string      `json:"color,omitempty"`
	DataFormula   string      `json:"data_formula,omitempty"`
	Derive        interface{} `json:"derive,omitempty"` // this is supposed to be a string but for CAQL statements it comes out as a boolean
	Hidden        bool        `json:"hidden,omitempty"`
	LegendFormula string      `json:"legend_formula,omitempty"`
	MetricName    string      `json:"metric_name,omitempty"`
	MetricType    string      `json:"metric_type,omitempty"`
	Name          string      `json:"name,omitempty"`
	Stack         int         `json:"stack,omitempty"`
}

// GraphGuide defines a guide
type GraphGuide struct {
	Color         string `json:"color,omitempty"`
	DataFormula   string `json:"data_formula,omitempty"`
	Hidden        bool   `json:"hidden,omitempty"`
	LegendFormula string `json:"legend_formula,omitempty"`
	Name          string `json:"name,omitempty"`
}

// GraphMetricCluster defines a metric cluster
type GraphMetricCluster struct {
	AggregateFunc string `json:"aggregation_function,omitempty"`
	Axis          string `json:"axis,omitempty"`
	DataFormula   string `json:"data_formula,omitempty"`
	Hidden        bool   `json:"hidden,omitempty"`
	LegendFormula string `json:"legend_formula,omitempty"`
	MetricCluster string `json:"metric_cluster,omitempty"`
	Name          string `json:"name,omitempty"`
	Stack         int    `json:"stack,omitempty"`
}

// OverlayDataOptions defines overlay options for data. Note, each overlay type requires
// a _subset_ of the options. See Graph API documentation (URL above) for details.
type OverlayDataOptions struct {
	Alerts        int    `json:"alerts,omitempty"`
	ArrayOutput   int    `json:"array_output,omitempty"`
	BasePeriod    int    `json:"base_period,omitempty"`
	Delay         int    `json:"delay,omitempty"`
	Extension     string `json:"extension,omitempty"`
	GraphTitle    string `json:"graph_title,omitempty"`
	GraphUUID     string `json:"graph_id,omitempty"`
	InPercent     string `json:"in_percent,omitempty"`
	Inverse       int    `json:"inverse,omitempty"`
	Method        string `json:"method,omitempty"`
	Model         string `json:"model,omitempty"`
	ModelEnd      string `json:"model_end,omitempty"`
	ModelRelative int    `json:"model_relative,omitempty"`
	Out           string `json:"out,omitempty"`
	Prequel       int    `json:"prequel,omitempty"`
	Presets       string `json:"presets,omitempty"`
	Quantiles     string `json:"quantiles,omitempty"`
	SeasonLength  int    `json:"season_length,omitempty"`
	Sensitivity   int    `json:"sensitivity,omitempty"`
	SingleValue   int    `json:"single_value,omitempty"`
	TargetPeriod  string `json:"target_period,omitempty"`
	TimeOffset    string `json:"time_offset,omitempty"`
	TimeShift     int    `json:"time_shift,omitempty"`
	Transform     string `json:"transform,omitempty"`
	Version       int    `json:"version,omitempty"`
	Window        int    `json:"window,omitempty"`
	XShift        string `json:"x_shift,omitempty"`
}

// OverlayUISpecs defines UI specs for overlay
type OverlayUISpecs struct {
	ID       string `json:"id,omitempty"`
	Z        int    `json:"z,omitempty"`
	Label    string `json:"label,omitempty"`
	Type     string `json:"type,omitempty"`
	Decouple bool   `json:"decouple,omitempty"`
}

// GraphOverlaySet defines overlays for graph
type GraphOverlaySet struct {
	ID       string             `json:"id,omitempty"`
	DataOpts OverlayDataOptions `json:"data_opts,omitempty"`
	UISpecs  OverlayUISpecs     `json:"ui_specs,omitempty"`
	Title    string             `json:"title,omitempty"`
}

// Graph definition
type Graph struct {
	CID            string                     `json:"_cid,omitempty"`
	AccessKeys     []GraphAccessKey           `json:"access_keys,omitempty"`
	Composites     []GraphComposite           `json:"composites,omitempty"`
	Datapoints     []GraphDatapoint           `json:"datapoints,omitempt"`
	Description    string                     `json:"description,omitempty"`
	Guides         []GraphGuide               `json:"guides,omitempty"`
	LineStyle      string                     `json:"line_style,omitempty"`
	LogLeftY       int                        `json:"logarithmitc_left_y,omitempty"`
	LogRightY      int                        `json:"logarithmitc_right_y,omitempty"`
	MaxLeftY       int                        `json:"max_left_y,omitempty"`
	MaxRightY      int                        `json:"max_right_y,omitempty"`
	MetricClusters []GraphMetricCluster       `json:"metric_clusters,omitempty"`
	MinLeftY       int                        `json:"min_left_y,omitempty"`
	MinRightY      int                        `json:"min_right_y,omitempty"`
	Notes          string                     `json:"notes,omitempty"`
	OverlaySets    map[string]GraphOverlaySet `json:"overlay_sets,omitempty"`
	Style          string                     `json:"style,omitempty"`
	Tags           []string                   `json:"tags,omitempty"`
	Title          string                     `json:"title,omitempty"`
}

const (
	baseGraphPath = "/graph"
	graphCIDRegex = "^" + baseGraphPath + "/[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{8,12}$"
)

// FetchGraph retrieves a graph definition
func (a *API) FetchGraph(cid CIDType) (*Graph, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid graph CID [none]")
	}

	graphCID := string(*cid)

	matched, err := regexp.MatchString(graphCIDRegex, graphCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid graph CID [%s]", graphCID)
	}

	result, err := a.Get(graphCID)
	if err != nil {
		return nil, err
	}

	graph := new(Graph)
	if err := json.Unmarshal(result, graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// FetchGraphs retrieves all graphs
func (a *API) FetchGraphs() (*[]Graph, error) {
	result, err := a.Get(baseGraphPath)
	if err != nil {
		return nil, err
	}

	var graphs []Graph
	if err := json.Unmarshal(result, &graphs); err != nil {
		return nil, err
	}

	return &graphs, nil
}

// UpdateGraph update graph definition
func (a *API) UpdateGraph(config *Graph) (*Graph, error) {

	if config == nil {
		return nil, fmt.Errorf("Invalid graph config [nil]")
	}

	graphCID := string(config.CID)

	if matched, err := regexp.MatchString(graphCIDRegex, graphCID); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid graph CID [%s]", graphCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(graphCID, cfg)
	if err != nil {
		return nil, err
	}

	graph := &Graph{}
	if err := json.Unmarshal(result, graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// CreateGraph create a new graph
func (a *API) CreateGraph(config *Graph) (*Graph, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid graph config [nil]")
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Post(baseGraphPath, cfg)
	if err != nil {
		return nil, err
	}

	graph := &Graph{}
	if err := json.Unmarshal(result, graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// DeleteGraph delete a graph
func (a *API) DeleteGraph(config *Graph) (bool, error) {
	if config == nil {
		return false, fmt.Errorf("Invalid graph config [nil]")
	}
	return a.DeleteGraphByCID(CIDType(&config.CID))
}

// DeleteGraphByCID delete a graph by cid
func (a *API) DeleteGraphByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid graph CID [none]")
	}

	graphCID := string(*cid)

	matched, err := regexp.MatchString(graphCIDRegex, graphCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid graph CID [%s]", graphCID)
	}

	_, err = a.Delete(graphCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GraphSearch returns list of graphs matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) GraphSearch(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Graph, error) {
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
		return a.FetchGraphs()
	}

	reqURL := url.URL{
		Path:     baseGraphPath,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var graphs []Graph
	if err := json.Unmarshal(result, &graphs); err != nil {
		return nil, err
	}

	return &graphs, nil
}
