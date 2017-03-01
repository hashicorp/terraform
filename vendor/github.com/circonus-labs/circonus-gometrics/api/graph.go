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
	Active         bool   `json:"active,omitempty"`           // boolean
	Height         uint   `json:"height,omitempty"`           // uint
	Key            string `json:"key,omitempty"`              // string
	Legend         bool   `json:"legend,omitempty"`           // boolean
	LockDate       bool   `json:"lock_date,omitempty"`        // boolean
	LockMode       string `json:"lock_mode,omitempty"`        // string
	LockRangeEnd   uint   `json:"lock_range_end,omitempty"`   // uint
	LockRangeStart uint   `json:"lock_range_start,omitempty"` // uint
	LockShowTimes  bool   `json:"lock_show_times,omitempty"`  // boolean
	LockZoom       string `json:"lock_zoom,omitempty"`        // string
	Nickname       string `json:"nickname,omitempty"`         // string
	Title          bool   `json:"title,omitempty"`            // boolean
	Width          uint   `json:"width,omitempty"`            // uint
	XLabels        bool   `json:"x_labels,omitempty"`         // boolean
	YLabels        bool   `json:"y_labels,omitempty"`         // boolean
}

// GraphComposite defines a composite
type GraphComposite struct {
	Axis          string  `json:"axis,omitempty"`           // string
	Color         string  `json:"color,omitempty"`          // string
	DataFormula   *string `json:"data_formula,omitempty"`   // string or null
	Hidden        bool    `json:"hidden,omitempty"`         // boolean
	LegendFormula *string `json:"legend_formula,omitempty"` // string or null
	Name          string  `json:"name,omitempty"`           // string
	Stack         *uint   `json:"stack,omitempty"`          // uint or null
}

// GraphDatapoint defines a datapoint
type GraphDatapoint struct {
	Alpha         *float64    `json:"alpha,string,omitempty"` // float64
	Axis          string      `json:"axis,omitempty"`         // string
	CAQL          *string     `json:"caql,omitempty"`         // string or null
	CheckID       uint        `json:"check_id,omitempty"`     // uint
	Color         *string     `json:"color,omitempty"`        // string
	DataFormula   *string     `json:"data_formula"`           // string or null
	Derive        interface{} `json:"derive,omitempty"`       // BUG doc: string, api: string or boolean(for caql statements)
	Hidden        bool        `json:"hidden"`                 // boolean
	LegendFormula *string     `json:"legend_formula"`         // string or null
	MetricName    string      `json:"metric_name,omitempty"`  // string
	MetricType    string      `json:"metric_type,omitempty"`  // string
	Name          string      `json:"name"`                   // string
	Stack         *uint       `json:"stack"`                  // uint or null
}

// GraphGuide defines a guide
type GraphGuide struct {
	Color         string  `json:"color,omitempty"`          // string
	DataFormula   *string `json:"data_formula,omitempty"`   // string or null
	Hidden        bool    `json:"hidden,omitempty"`         // boolean
	LegendFormula *string `json:"legend_formula,omitempty"` // string or null
	Name          string  `json:"name,omitempty"`           // string
}

// GraphMetricCluster defines a metric cluster
type GraphMetricCluster struct {
	AggregateFunc string  `json:"aggregate_function,omitempty"` // string
	Axis          string  `json:"axis,omitempty"`               // string
	Color         *string `json:"color,omitempty"`              // string
	DataFormula   *string `json:"data_formula"`                 // string or null
	Hidden        bool    `json:"hidden"`                       // boolean
	LegendFormula *string `json:"legend_formula"`               // string or null
	MetricCluster string  `json:"metric_cluster,omitempty"`     // string
	Name          string  `json:"name,omitempty"`               // string
	Stack         *uint   `json:"stack"`                        // uint or null
}

// OverlayDataOptions defines overlay options for data. Note, each overlay type requires
// a _subset_ of the options. See Graph API documentation (URL above) for details.
type OverlayDataOptions struct {
	Alerts        *int   `json:"alerts,string,omitempty"`         // int encoded as string BUG doc: numeric, api: string
	ArrayOutput   *int   `json:"array_output,string,omitempty"`   // int encoded as string BUG doc: numeric, api: string
	BasePeriod    *int   `json:"base_period,string,omitempty"`    // int encoded as string BUG doc: numeric, api: string
	Delay         *int   `json:"delay,string,omitempty"`          // int encoded as string BUG doc: numeric, api: string
	Extension     string `json:"extension,omitempty"`             // string
	GraphTitle    string `json:"graph_title,omitempty"`           // string
	GraphUUID     string `json:"graph_id,omitempty"`              // string
	InPercent     *bool  `json:"in_percent,string,omitempty"`     // boolean encoded as string BUG doc: boolean, api: string
	Inverse       *int   `json:"inverse,string,omitempty"`        // int encoded as string BUG doc: numeric, api: string
	Method        string `json:"method,omitempty"`                // string
	Model         string `json:"model,omitempty"`                 // string
	ModelEnd      string `json:"model_end,omitempty"`             // string
	ModelPeriod   string `json:"model_period,omitempty"`          // string
	ModelRelative *int   `json:"model_relative,string,omitempty"` // int encoded as string BUG doc: numeric, api: string
	Out           string `json:"out,omitempty"`                   // string
	Prequel       string `json:"prequel,omitempty"`               // string
	Presets       string `json:"presets,omitempty"`               // string
	Quantiles     string `json:"quantiles,omitempty"`             // string
	SeasonLength  *int   `json:"season_length,string,omitempty"`  // int encoded as string BUG doc: numeric, api: string
	Sensitivity   *int   `json:"sensitivity,string,omitempty"`    // int encoded as string BUG doc: numeric, api: string
	SingleValue   *int   `json:"single_value,string,omitempty"`   // int encoded as string BUG doc: numeric, api: string
	TargetPeriod  string `json:"target_period,omitempty"`         // string
	TimeOffset    string `json:"time_offset,omitempty"`           // string
	TimeShift     *int   `json:"time_shift,string,omitempty"`     // int encoded as string BUG doc: numeric, api: string
	Transform     string `json:"transform,omitempty"`             // string
	Version       *int   `json:"version,string,omitempty"`        // int encoded as string BUG doc: numeric, api: string
	Window        *int   `json:"window,string,omitempty"`         // int encoded as string BUG doc: numeric, api: string
	XShift        string `json:"x_shift,omitempty"`               // string
}

// OverlayUISpecs defines UI specs for overlay
type OverlayUISpecs struct {
	Decouple bool   `json:"decouple,omitempty"` // boolean
	ID       string `json:"id,omitempty"`       // string
	Label    string `json:"label,omitempty"`    // string
	Type     string `json:"type,omitempty"`     // string
	Z        *int   `json:"z,string,omitempty"` // int encoded as string BUG doc: numeric, api: string
}

// GraphOverlaySet defines overlays for graph
type GraphOverlaySet struct {
	DataOpts OverlayDataOptions `json:"data_opts,omitempty"` // OverlayDataOptions
	ID       string             `json:"id,omitempty"`        // string
	Title    string             `json:"title,omitempty"`     // string
	UISpecs  OverlayUISpecs     `json:"ui_specs,omitempty"`  // OverlayUISpecs
}

// Graph defines a graph. See https://login.circonus.com/resources/api/calls/graph for more information.
type Graph struct {
	AccessKeys     []GraphAccessKey            `json:"access_keys,omitempty"`                // [] len >= 0
	CID            string                      `json:"_cid,omitempty"`                       // string
	Composites     []GraphComposite            `json:"composites,omitempty"`                 // [] len >= 0
	Datapoints     []GraphDatapoint            `json:"datapoints,omitempt"`                  // [] len >= 0
	Description    string                      `json:"description,omitempty"`                // string
	Guides         []GraphGuide                `json:"guides,omitempty"`                     // [] len >= 0
	LineStyle      *string                     `json:"line_style"`                           // string or null
	LogLeftY       *int                        `json:"logarithmic_left_y,string,omitempty"`  // int encoded as string or null BUG doc: number (not string)
	LogRightY      *int                        `json:"logarithmic_right_y,string,omitempty"` // int encoded as string or null BUG doc: number (not string)
	MaxLeftY       *float64                    `json:"max_left_y,string,omitempty"`          // float64 encoded as string or null BUG doc: number (not string)
	MaxRightY      *float64                    `json:"max_right_y,string,omitempty"`         // float64 encoded as string or null BUG doc: number (not string)
	MetricClusters []GraphMetricCluster        `json:"metric_clusters,omitempty"`            // [] len >= 0
	MinLeftY       *float64                    `json:"min_left_y,string,omitempty"`          // float64 encoded as string or null BUG doc: number (not string)
	MinRightY      *float64                    `json:"min_right_y,string,omitempty"`         // float64 encoded as string or null BUG doc: number (not string)
	Notes          *string                     `json:"notes,omitempty"`                      // string or null
	OverlaySets    *map[string]GraphOverlaySet `json:"overlay_sets,omitempty"`               // GroupOverLaySets or null
	Style          *string                     `json:"style"`                                // string or null
	Tags           []string                    `json:"tags,omitempty"`                       // [] len >= 0
	Title          string                      `json:"title,omitempty"`                      // string
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
