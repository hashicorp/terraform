// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Dashboard API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/dashboard

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// DashboardGridLayout defines layout
type DashboardGridLayout struct {
	Height uint `json:"height"`
	Width  uint `json:"width"`
}

// DashboardAccessConfig defines access config
type DashboardAccessConfig struct {
	BlackDash           bool   `json:"black_dash,omitempty"`
	Enabled             bool   `json:"enabled,omitempty"`
	Fullscreen          bool   `json:"fullscreen,omitempty"`
	FullscreenHideTitle bool   `json:"fullscreen_hide_title,omitempty"`
	Nickname            string `json:"nickname,omitempty"`
	ScaleText           bool   `json:"scale_text,omitempty"`
	SharedID            string `json:"shared_id,omitempty"`
	TextSize            uint   `json:"text_size,omitempty"`
}

// DashboardOptions defines options
type DashboardOptions struct {
	AccessConfigs       []DashboardAccessConfig `json:"access_configs,omitempty"`
	FullscreenHideTitle bool                    `json:"fullscreen_hide_title,omitempty"`
	HideGrid            bool                    `json:"hide_grid,omitempty"`
	Linkages            [][]string              `json:"linkages,omitempty"`
	ScaleText           bool                    `json:"scale_text,omitempty"`
	TextSize            uint                    `json:"text_size,omitempty"`
}

// ChartTextWidgetDatapoint defines datapoints for charts
type ChartTextWidgetDatapoint struct {
	AccountID    string `json:"account_id,omitempty"`     // metric cluster, metric
	CheckID      uint   `json:"_check_id,omitempty"`      // metric
	ClusterID    uint   `json:"cluster_id,omitempty"`     // metric cluster
	ClusterTitle string `json:"_cluster_title,omitempty"` // metric cluster
	Label        string `json:"label,omitempty"`          // metric
	Label2       string `json:"_label,omitempty"`         // metric cluster
	Metric       string `json:"metric,omitempty"`         // metric
	MetricType   string `json:"_metric_type,omitempty"`   // metric
	NumericOnly  bool   `json:"numeric_only,omitempty"`   // metric cluster
}

// ChartWidgetDefinitionLegend defines chart widget definition legend
type ChartWidgetDefinitionLegend struct {
	Show bool   `json:"show,omitempty"`
	Type string `json:"type,omitempty"`
}

// ChartWidgetWedgeLabels defines chart widget wedge labels
type ChartWidgetWedgeLabels struct {
	OnChart  bool `json:"on_chart,omitempty"`
	ToolTips bool `json:"tooltips,omitempty"`
}

// ChartWidgetWedgeValues defines chart widget wedge values
type ChartWidgetWedgeValues struct {
	Angle string `json:"angle,omitempty"`
	Color string `json:"color,omitempty"`
	Show  bool   `json:"show,omitempty"`
}

// ChartWidgtDefinition defines chart widget definition
type ChartWidgtDefinition struct {
	Datasource        string                      `json:"datasource,omitempty"`
	Derive            string                      `json:"derive,omitempty"`
	DisableAutoformat bool                        `json:"disable_autoformat,omitempty"`
	Formula           string                      `json:"formula,omitempty"`
	Legend            ChartWidgetDefinitionLegend `json:"legend,omitempty"`
	Period            uint                        `json:"period,omitempty"`
	PopOnHover        bool                        `json:"pop_onhover,omitempty"`
	WedgeLabels       ChartWidgetWedgeLabels      `json:"wedge_labels,omitempty"`
	WedgeValues       ChartWidgetWedgeValues      `json:"wedge_values,omitempty"`
}

// ForecastGaugeWidgetThresholds defines forecast widget thresholds
type ForecastGaugeWidgetThresholds struct {
	Colors []string `json:"colors,omitempty"` // forecasts, gauges
	Flip   bool     `json:"flip,omitempty"`   // gauges
	Values []string `json:"values,omitempty"` // forecasts, gauges
}

// StatusWidgetAgentStatusSettings defines agent status settings
type StatusWidgetAgentStatusSettings struct {
	Search         string `json:"search,omitempty"`
	ShowAgentTypes string `json:"show_agent_types,omitempty"`
	ShowContact    bool   `json:"show_contact,omitempty"`
	ShowFeeds      bool   `json:"show_feeds,omitempty"`
	ShowSetup      bool   `json:"show_setup,omitempty"`
	ShowSkew       bool   `json:"show_skew,omitempty"`
	ShowUpdates    bool   `json:"show_updates,omitempty"`
}

// StatusWidgetHostStatusSettings defines host status settings
type StatusWidgetHostStatusSettings struct {
	LayoutStyle  string   `json:"layout_style,omitempty"`
	Search       string   `json:"search,omitempty"`
	SortBy       string   `json:"sort_by,omitempty"`
	TagFilterSet []string `json:"tag_filter_set,omitempty"`
}

// DashboardWidgetSettings defines settings specific to widget
type DashboardWidgetSettings struct {
	AccountID           string                          `json:"account_id,omitempty"`            // alerts, clusters, gauges, graphs, lists, status
	Acknowledged        string                          `json:"acknowledged,omitempty"`          // alerts
	AgentStatusSettings StatusWidgetAgentStatusSettings `json:"agent_status_settings,omitempty"` // status
	Algorithm           string                          `json:"algorithm,omitempty"`             // clusters
	Autoformat          bool                            `json:"autoformat,omitempty"`            // text
	BodyFormat          string                          `json:"body_format,omitempty"`           // text
	ChartType           string                          `json:"chart_type,omitempty"`            // charts
	CheckUUID           string                          `json:"check_uuid,omitempty"`            // gauges
	Cleared             string                          `json:"cleared,omitempty"`               // alerts
	ClusterID           uint                            `json:"cluster_id,omitempty"`            // clusters
	ClusterName         string                          `json:"cluster_name,omitempty"`          // clusters
	ContactGroups       []uint                          `json:"contact_groups,omitempty"`        // alerts
	ContentType         string                          `json:"content_type,omitempty"`          // status
	Datapoints          []ChartTextWidgetDatapoint      `json:"datapoints,omitempty"`            // charts, text
	DateWindow          string                          `json:"date_window,omitempty"`           // graphs
	Definition          ChartWidgtDefinition            `json:"definition,omitempty"`            // charts
	Dependents          string                          `json:"dependents,omitempty"`            // alerts
	DisableAutoformat   bool                            `json:"disable_autoformat,omitempty"`    // gauges
	Display             string                          `json:"display,omitempty"`               // alerts
	Format              string                          `json:"format,omitempty"`                // forecasts
	Formula             string                          `json:"formula,omitempty"`               // gauges
	GraphUUID           string                          `json:"graph_id,omitempty"`              // graphs
	HideXAxis           bool                            `json:"hide_xaxis,omitempty"`            // graphs
	HideYAxis           bool                            `json:"hide_yaxis,omitempty"`            // graphs
	HostStatusSettings  StatusWidgetHostStatusSettings  `json:"host_status_settings,omitempty"`  // status
	KeyInline           bool                            `json:"key_inline,omitempty"`            // graphs
	KeyLoc              string                          `json:"key_loc,omitempty"`               // graphs
	KeySize             uint                            `json:"key_size,omitempty"`              // graphs
	KeyWrap             bool                            `json:"key_wrap,omitempty"`              // graphs
	Label               string                          `json:"label,omitempty"`                 // graphs
	Layout              string                          `json:"layout,omitempty"`                // clusters
	Limit               uint                            `json:"limit,omitempty"`                 // lists
	Maintenance         string                          `json:"maintenance,omitempty"`           // alerts
	Markup              string                          `json:"markup,omitempty"`                // html
	MetricDisplayName   string                          `json:"metric_display_name,omitempty"`   // gauges
	MetricName          string                          `json:"metric_name,omitempty"`           // gauges
	MinAge              string                          `json:"min_age,omitempty"`               // alerts
	OffHours            []uint                          `json:"off_hours,omitempty"`             // alerts
	OverlaySetID        string                          `json:"overlay_set_id,omitempty"`        // graphs
	Period              uint                            `json:"period,omitempty"`                // gauges, text, graphs
	RangeHigh           int                             `json:"range_high,omitempty"`            // gauges
	RangeLow            int                             `json:"range_low,omitempty"`             // gauges
	Realtime            bool                            `json:"realtime,omitempty"`              // graphs
	ResourceLimit       string                          `json:"resource_limit,omitempty"`        // forecasts
	ResourceUsage       string                          `json:"resource_usage,omitempty"`        // forecasts
	Search              string                          `json:"search,omitempty"`                // alerts, lists
	Severity            string                          `json:"severity,omitempty"`              // alerts
	ShowFlags           bool                            `json:"show_flags,omitempty"`            // graphs
	Size                string                          `json:"size,omitempty"`                  // clusters
	TagFilterSet        []string                        `json:"tag_filter_set,omitempty"`        // alerts
	Threshold           float32                         `json:"threshold,omitempty"`             // clusters
	Thresholds          ForecastGaugeWidgetThresholds   `json:"thresholds,omitempty"`            // forecasts, gauges
	TimeWindow          string                          `json:"time_window,omitempty"`           // alerts
	Title               string                          `json:"title,omitempty"`                 // alerts, charts, forecasts, gauges, html
	TitleFormat         string                          `json:"title_format,omitempty"`          // text
	Trend               string                          `json:"trend,omitempty"`                 // forecasts
	Type                string                          `json:"type,omitempty"`                  // gauges, lists
	UseDefault          bool                            `json:"use_default,omitempty"`           // text
	ValueType           string                          `json:"value_type,omitempty"`            // gauges, text
	WeekDays            []string                        `json:"weekdays,omitempty"`              // alerts
}

// DashboardWidget defines widget
type DashboardWidget struct {
	Active   bool                    `json:"active"`
	Height   uint                    `json:"height"`
	Name     string                  `json:"name"`
	Origin   string                  `json:"origin"`
	Settings DashboardWidgetSettings `json:"settings"`
	Type     string                  `json:"type"`
	WidgetID string                  `json:"widget_id"`
	Width    uint                    `json:"width"`
}

// Dashboard defines a dashboard. See https://login.circonus.com/resources/api/calls/dashboard for more information.
type Dashboard struct {
	AccountDefault bool                `json:"account_default"`
	Active         bool                `json:"_active,omitempty"`
	CID            string              `json:"_cid,omitempty"`
	Created        uint                `json:"_created,omitempty"`
	CreatedBy      string              `json:"_created_by,omitempty"`
	GridLayout     DashboardGridLayout `json:"grid_layout"`
	LastModified   uint                `json:"_last_modified,omitempty"`
	Options        DashboardOptions    `json:"options"`
	Shared         bool                `json:"shared"`
	Title          string              `json:"title"`
	UUID           string              `json:"_dashboard_uuid,omitempty"`
	Widgets        []DashboardWidget   `json:"widgets"`
}

// NewDashboard returns a new Dashboard (with defaults, if applicable)
func NewDashboard() *Dashboard {
	return &Dashboard{}
}

// FetchDashboard retrieves dashboard with passed cid.
func (a *API) FetchDashboard(cid CIDType) (*Dashboard, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid dashboard CID [none]")
	}

	dashboardCID := string(*cid)

	matched, err := regexp.MatchString(config.DashboardCIDRegex, dashboardCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid dashboard CID [%s]", dashboardCID)
	}

	result, err := a.Get(string(*cid))
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch dashboard, received JSON: %s", string(result))
	}

	dashboard := new(Dashboard)
	if err := json.Unmarshal(result, dashboard); err != nil {
		return nil, err
	}

	return dashboard, nil
}

// FetchDashboards retrieves all dashboards available to the API Token.
func (a *API) FetchDashboards() (*[]Dashboard, error) {
	result, err := a.Get(config.DashboardPrefix)
	if err != nil {
		return nil, err
	}

	var dashboards []Dashboard
	if err := json.Unmarshal(result, &dashboards); err != nil {
		return nil, err
	}

	return &dashboards, nil
}

// UpdateDashboard updates passed dashboard.
func (a *API) UpdateDashboard(cfg *Dashboard) (*Dashboard, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid dashboard config [nil]")
	}

	dashboardCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.DashboardCIDRegex, dashboardCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid dashboard CID [%s]", dashboardCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update dashboard, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(dashboardCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	dashboard := &Dashboard{}
	if err := json.Unmarshal(result, dashboard); err != nil {
		return nil, err
	}

	return dashboard, nil
}

// CreateDashboard creates a new dashboard.
func (a *API) CreateDashboard(cfg *Dashboard) (*Dashboard, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid dashboard config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create dashboard, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.DashboardPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	dashboard := &Dashboard{}
	if err := json.Unmarshal(result, dashboard); err != nil {
		return nil, err
	}

	return dashboard, nil
}

// DeleteDashboard deletes passed dashboard.
func (a *API) DeleteDashboard(cfg *Dashboard) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid dashboard config [nil]")
	}
	return a.DeleteDashboardByCID(CIDType(&cfg.CID))
}

// DeleteDashboardByCID deletes dashboard with passed cid.
func (a *API) DeleteDashboardByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid dashboard CID [none]")
	}

	dashboardCID := string(*cid)

	matched, err := regexp.MatchString(config.DashboardCIDRegex, dashboardCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid dashboard CID [%s]", dashboardCID)
	}

	_, err = a.Delete(dashboardCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchDashboards returns dashboards matching the specified
// search query and/or filter. If nil is passed for both parameters
// all dashboards will be returned.
func (a *API) SearchDashboards(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Dashboard, error) {
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
		return a.FetchDashboards()
	}

	reqURL := url.URL{
		Path:     config.DashboardPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var dashboards []Dashboard
	if err := json.Unmarshal(result, &dashboards); err != nil {
		return nil, err
	}

	return &dashboards, nil
}
