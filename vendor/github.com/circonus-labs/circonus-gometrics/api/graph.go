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

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// GraphAccessKey defines an access key for a graph
type GraphAccessKey struct {
	Active         bool   `json:"active,omitempty"`
	Height         uint   `json:"height,omitempty"`
	Key            string `json:"key,omitempty"`
	Legend         bool   `json:"legend,omitempty"`
	LockDate       bool   `json:"lock_date,omitempty"`
	LockMode       string `json:"lock_mode,omitempty"`
	LockRangeEnd   uint   `json:"lock_range_end,omitempty"`
	LockRangeStart uint   `json:"lock_range_start,omitempty"`
	LockShowTimes  bool   `json:"lock_show_times,omitempty"`
	LockZoom       string `json:"lock_zoom,omitempty"`
	Nickname       string `json:"nickname,omitempty"`
	Title          bool   `json:"title,omitempty"`
	Width          uint   `json:"width,omitempty"`
	XLabels        bool   `json:"x_labels,omitempty"`
	YLabels        bool   `json:"y_labels,omitempty"`
}

// GraphComposite defines a composite
type GraphComposite struct {
	Axis          string  `json:"axis,omitempty"`
	Color         string  `json:"color,omitempty"`
	DataFormula   *string `json:"data_formula,omitempty"` // null or string
	Hidden        bool    `json:"hidden,omitempty"`
	LegendFormula *string `json:"legend_formula,omitempty"` // null or string
	Name          string  `json:"name,omitempty"`
	Stack         *uint   `json:"stack,omitempty"` // null or uint
}

// GraphDatapoint defines a datapoint
type GraphDatapoint struct {
	Alpha         string      `json:"alpha,omitempty"`
	Axis          string      `json:"axis,omitempty"`
	CAQL          *string     `json:"caql,omitempty"` // null or string
	CheckID       uint        `json:"check_id,omitempty"`
	Color         string      `json:"color,omitempty"`
	DataFormula   *string     `json:"data_formula,omitempty"` // null or string
	Derive        interface{} `json:"derive,omitempty"`       // BUG this is supposed to be a string but for CAQL statements it comes out as a boolean
	Hidden        bool        `json:"hidden,omitempty"`
	LegendFormula string      `json:"legend_formula,omitempty"`
	MetricName    string      `json:"metric_name,omitempty"`
	MetricType    string      `json:"metric_type,omitempty"`
	Name          string      `json:"name,omitempty"`
	Stack         *uint       `json:"stack,omitempty"` // null or uint
}

// GraphGuide defines a guide
type GraphGuide struct {
	Color         string  `json:"color,omitempty"`
	DataFormula   *string `json:"data_formula,omitempty"` // null or string
	Hidden        bool    `json:"hidden,omitempty"`
	LegendFormula *string `json:"legend_formula,omitempty"` // null or string
	Name          string  `json:"name,omitempty"`
}

// GraphMetricCluster defines a metric cluster
type GraphMetricCluster struct {
	AggregateFunc string  `json:"aggregation_function,omitempty"`
	Axis          string  `json:"axis,omitempty"`
	DataFormula   *string `json:"data_formula,omitempty"` // null or string
	Hidden        bool    `json:"hidden,omitempty"`
	LegendFormula *string `json:"legend_formula,omitempty"` // null or string
	MetricCluster string  `json:"metric_cluster,omitempty"`
	Name          string  `json:"name,omitempty"`
	Stack         *uint   `json:"stack,omitempty"` // null or uint
}

// OverlayDataOptions defines overlay options for data. Note, each overlay type requires
// a _subset_ of the options. See Graph API documentation (URL above) for details.
type OverlayDataOptions struct {
	Alerts        uint   `json:"alerts,omitempty"`
	ArrayOutput   uint   `json:"array_output,omitempty"`
	BasePeriod    uint   `json:"base_period,omitempty"`
	Delay         uint   `json:"delay,omitempty"`
	Extension     string `json:"extension,omitempty"`
	GraphTitle    string `json:"graph_title,omitempty"`
	GraphUUID     string `json:"graph_id,omitempty"`
	InPercent     string `json:"in_percent,omitempty"`
	Inverse       uint   `json:"inverse,omitempty"`
	Method        string `json:"method,omitempty"`
	Model         string `json:"model,omitempty"`
	ModelEnd      string `json:"model_end,omitempty"`
	ModelRelative uint   `json:"model_relative,omitempty"`
	Out           string `json:"out,omitempty"`
	Prequel       uint   `json:"prequel,omitempty"`
	Presets       string `json:"presets,omitempty"`
	Quantiles     string `json:"quantiles,omitempty"`
	SeasonLength  uint   `json:"season_length,omitempty"`
	Sensitivity   uint   `json:"sensitivity,omitempty"`
	SingleValue   uint   `json:"single_value,omitempty"`
	TargetPeriod  string `json:"target_period,omitempty"`
	TimeOffset    string `json:"time_offset,omitempty"`
	TimeShift     int    `json:"time_shift,omitempty"`
	Transform     string `json:"transform,omitempty"`
	Version       uint   `json:"version,omitempty"`
	Window        uint   `json:"window,omitempty"`
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

// Graph defines a graph. See https://login.circonus.com/resources/api/calls/graph for more information.
type Graph struct {
	CID            string                      `json:"_cid,omitempty"`
	AccessKeys     []GraphAccessKey            `json:"access_keys,omitempty"`
	Composites     []GraphComposite            `json:"composites,omitempty"`
	Datapoints     []GraphDatapoint            `json:"datapoints,omitempt"`
	Description    string                      `json:"description,omitempty"`
	Guides         []GraphGuide                `json:"guides,omitempty"`
	LineStyle      string                      `json:"line_style,omitempty"`
	LogLeftY       int                         `json:"logarithmitc_left_y,omitempty"`  // BUG doc: number, comes as null|string
	LogRightY      int                         `json:"logarithmitc_right_y,omitempty"` // BUG doc: number, comes as null|string
	MaxLeftY       *string                     `json:"max_left_y,omitempty"`           // BUG doc: number, comes as null|string
	MaxRightY      *string                     `json:"max_right_y,omitempty"`          // BUG doc: number, comes as null|string
	MetricClusters []GraphMetricCluster        `json:"metric_clusters,omitempty"`
	MinLeftY       *string                     `json:"min_left_y,omitempty"`  // BUG doc: number, comes as null|string
	MinRightY      *string                     `json:"min_right_y,omitempty"` // BUG doc: number, comes as null|string
	Notes          string                      `json:"notes,omitempty"`
	OverlaySets    *map[string]GraphOverlaySet `json:"overlay_sets,omitempty"` // null or overlay sets object
	Style          string                      `json:"style,omitempty"`
	Tags           []string                    `json:"tags,omitempty"`
	Title          string                      `json:"title,omitempty"`
}

// NewGraph returns a Graph (with defaults, if applicable)
func NewGraph() *Graph {
	return &Graph{}
}

// FetchGraph retrieves graph with passed cid.
func (a *API) FetchGraph(cid CIDType) (*Graph, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid graph CID [none]")
	}

	graphCID := string(*cid)

	matched, err := regexp.MatchString(config.GraphCIDRegex, graphCID)
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
	if a.Debug {
		a.Log.Printf("[DEBUG] fetch graph, received JSON: %s", string(result))
	}

	graph := new(Graph)
	if err := json.Unmarshal(result, graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// FetchGraphs retrieves all graphs available to the API Token.
func (a *API) FetchGraphs() (*[]Graph, error) {
	result, err := a.Get(config.GraphPrefix)
	if err != nil {
		return nil, err
	}

	var graphs []Graph
	if err := json.Unmarshal(result, &graphs); err != nil {
		return nil, err
	}

	return &graphs, nil
}

// UpdateGraph updates passed graph.
func (a *API) UpdateGraph(cfg *Graph) (*Graph, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid graph config [nil]")
	}

	graphCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.GraphCIDRegex, graphCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid graph CID [%s]", graphCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update graph, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(graphCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	graph := &Graph{}
	if err := json.Unmarshal(result, graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// CreateGraph creates a new graph.
func (a *API) CreateGraph(cfg *Graph) (*Graph, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid graph config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update graph, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.GraphPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	graph := &Graph{}
	if err := json.Unmarshal(result, graph); err != nil {
		return nil, err
	}

	return graph, nil
}

// DeleteGraph deletes passed graph.
func (a *API) DeleteGraph(cfg *Graph) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid graph config [nil]")
	}
	return a.DeleteGraphByCID(CIDType(&cfg.CID))
}

// DeleteGraphByCID deletes graph with passed cid.
func (a *API) DeleteGraphByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid graph CID [none]")
	}

	graphCID := string(*cid)

	matched, err := regexp.MatchString(config.GraphCIDRegex, graphCID)
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

// SearchGraphs returns graphs matching the specified search query
// and/or filter. If nil is passed for both parameters all graphs
// will be returned.
func (a *API) SearchGraphs(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Graph, error) {
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
		Path:     config.GraphPrefix,
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
