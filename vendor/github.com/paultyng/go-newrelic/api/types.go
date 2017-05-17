package api

import "errors"

var (
	// ErrNotFound is returned when the resource was not found in New Relic.
	ErrNotFound = errors.New("newrelic: Resource not found")
)

// LabelLinks represents external references on the Label.
type LabelLinks struct {
	Applications []int `json:"applications"`
	Servers      []int `json:"servers"`
}

// Label represents a New Relic label.
type Label struct {
	Key      string     `json:"key,omitempty"`
	Category string     `json:"category,omitempty"`
	Name     string     `json:"name,omitempty"`
	Links    LabelLinks `json:"links,omitempty"`
}

// AlertPolicy represents a New Relic alert policy.
type AlertPolicy struct {
	ID                 int    `json:"id,omitempty"`
	IncidentPreference string `json:"incident_preference,omitempty"`
	Name               string `json:"name,omitempty"`
	CreatedAt          int    `json:"created_at,omitempty"`
	UpdatedAt          int    `json:"updated_at,omitempty"`
}

// AlertConditionUserDefined represents user defined metrics for the New Relic alert condition.
type AlertConditionUserDefined struct {
	Metric        string `json:"metric,omitempty"`
	ValueFunction string `json:"value_function,omitempty"`
}

// AlertConditionTerm represents the terms of a New Relic alert condition.
type AlertConditionTerm struct {
	Duration     int     `json:"duration,string,omitempty"`
	Operator     string  `json:"operator,omitempty"`
	Priority     string  `json:"priority,omitempty"`
	Threshold    float64 `json:"threshold,string,omitempty"`
	TimeFunction string  `json:"time_function,omitempty"`
}

// AlertCondition represents a New Relic alert condition.
// TODO: custom unmarshal entities to ints?
// TODO: handle unmarshaling .75 for float (not just 0.75)
type AlertCondition struct {
	PolicyID    int                       `json:"-"`
	ID          int                       `json:"id,omitempty"`
	Type        string                    `json:"type,omitempty"`
	Name        string                    `json:"name,omitempty"`
	Enabled     bool                      `json:"enabled,omitempty"`
	Entities    []string                  `json:"entities,omitempty"`
	Metric      string                    `json:"metric,omitempty"`
	RunbookURL  string                    `json:"runbook_url,omitempty"`
	Terms       []AlertConditionTerm      `json:"terms,omitempty"`
	UserDefined AlertConditionUserDefined `json:"user_defined,omitempty"`
	Scope       string                    `json:"condition_scope,omitempty"`
}

// AlertChannelLinks represent the links between policies and alert channels
type AlertChannelLinks struct {
	PolicyIDs []int `json:"policy_ids,omitempty"`
}

// AlertChannel represents a New Relic alert notification channel
type AlertChannel struct {
	ID            int                    `json:"id,omitempty"`
	Name          string                 `json:"name,omitempty"`
	Type          string                 `json:"type,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Links         AlertChannelLinks      `json:"links,omitempty"`
}

// ApplicationSummary represents performance information about a New Relic application.
type ApplicationSummary struct {
	ResponseTime            float64 `json:"response_time"`
	Throughput              float64 `json:"throughput"`
	ErrorRate               float64 `json:"error_rate"`
	ApdexTarget             float64 `json:"apdex_target"`
	ApdexScore              float64 `json:"apdex_score"`
	HostCount               int     `json:"host_count"`
	InstanceCount           int     `json:"instance_count"`
	ConcurrentInstanceCount int     `json:"concurrent_instance_count"`
}

// ApplicationEndUserSummary represents performance information about a New Relic application.
type ApplicationEndUserSummary struct {
	ResponseTime float64 `json:"response_time"`
	Throughput   float64 `json:"throughput"`
	ApdexTarget  float64 `json:"apdex_target"`
	ApdexScore   float64 `json:"apdex_score"`
}

// ApplicationSettings represents some of the settings of a New Relic application.
type ApplicationSettings struct {
	AppApdexThreshold        float64 `json:"app_apdex_threshold,omitempty"`
	EndUserApdexThreshold    float64 `json:"end_user_apdex_threshold,omitempty"`
	EnableRealUserMonitoring bool    `json:"enable_real_user_monitoring,omitempty"`
	UseServerSideConfig      bool    `json:"use_server_side_config,omitempty"`
}

// ApplicationLinks represents all the links for a New Relic application.
type ApplicationLinks struct {
	ServerIDs     []int `json:"servers,omitempty"`
	HostIDs       []int `json:"application_hosts,omitempty"`
	InstanceIDs   []int `json:"application_instances,omitempty"`
	AlertPolicyID int   `json:"alert_policy"`
}

// Application represents information about a New Relic application.
type Application struct {
	ID             int                       `json:"id,omitempty"`
	Name           string                    `json:"name,omitempty"`
	Language       string                    `json:"language,omitempty"`
	HealthStatus   string                    `json:"health_status,omitempty"`
	Reporting      bool                      `json:"reporting,omitempty"`
	LastReportedAt string                    `json:"last_reported_at,omitempty"`
	Summary        ApplicationSummary        `json:"application_summary,omitempty"`
	EndUserSummary ApplicationEndUserSummary `json:"end_user_summary,omitempty"`
	Settings       ApplicationSettings       `json:"settings,omitempty"`
	Links          ApplicationLinks          `json:"links,omitempty"`
}

// PluginDetails represents information about a New Relic plugin.
type PluginDetails struct {
	Description           int    `json:"description"`
	IsPublic              string `json:"is_public"`
	CreatedAt             string `json:"created_at,omitempty"`
	UpdatedAt             string `json:"updated_at,omitempty"`
	LastPublishedAt       string `json:"last_published_at,omitempty"`
	HasUnpublishedChanges bool   `json:"has_unpublished_changes"`
	BrandingImageURL      string `json:"branding_image_url"`
	UpgradedAt            string `json:"upgraded_at,omitempty"`
	ShortName             string `json:"short_name"`
	PublisherAboutURL     string `json:"publisher_about_url"`
	PublisherSupportURL   string `json:"publisher_support_url"`
	DownloadURL           string `json:"download_url"`
	FirstEditedAt         string `json:"first_edited_at,omitempty"`
	LastEditedAt          string `json:"last_edited_at,omitempty"`
	FirstPublishedAt      string `json:"first_published_at,omitempty"`
	PublishedVersion      string `json:"published_version"`
}

// MetricThreshold represents the different thresholds for a metric in an alert.
type MetricThreshold struct {
	Caution  float64 `json:"caution"`
	Critical float64 `json:"critical"`
}

// MetricValue represents the observed value of a metric.
type MetricValue struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

// MetricTimeslice represents the values of a metric over a given time.
type MetricTimeslice struct {
	From   string                 `json:"from,omitempty"`
	To     string                 `json:"to,omitempty"`
	Values map[string]interface{} `json:"values,omitempty"`
}

// Metric represents data for a specific metric.
type Metric struct {
	Name       string            `json:"name"`
	Timeslices []MetricTimeslice `json:"timeslices"`
}

// SummaryMetric represents summary information for a specific metric.
type SummaryMetric struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Metric        string          `json:"metric"`
	ValueFunction string          `json:"value_function"`
	Thresholds    MetricThreshold `json:"thresholds"`
	Values        MetricValue     `json:"values"`
}

// Plugin represents information about a New Relic plugin.
type Plugin struct {
	ID                  int             `json:"id"`
	Name                string          `json:"name,omitempty"`
	GUID                string          `json:"guid,omitempty"`
	Publisher           string          `json:"publisher,omitempty"`
	ComponentAgentCount int             `json:"component_agent_count"`
	Details             PluginDetails   `json:"details"`
	SummaryMetrics      []SummaryMetric `json:"summary_metrics"`
}

// Component represnets information about a New Relic component.
type Component struct {
	ID             int             `json:"id"`
	Name           string          `json:"name,omitempty"`
	HealthStatus   string          `json:"health_status,omitempty"`
	SummaryMetrics []SummaryMetric `json:"summary_metrics"`
}

// ComponentMetric represents metric information for a specific component.
type ComponentMetric struct {
	Name   string   `json:"name,omitempty"`
	Values []string `json:"values"`
}
