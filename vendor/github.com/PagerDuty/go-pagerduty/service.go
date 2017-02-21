package pagerduty

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// Integration is an endpoint (like Nagios, email, or an API call) that generates events, which are normalized and de-duplicated by PagerDuty to create incidents.
type Integration struct {
	APIObject
	Name             string     `json:"name,omitempty"`
	Service          *APIObject `json:"service,omitempty"`
	CreatedAt        string     `json:"created_at,omitempty"`
	Vendor           *APIObject `json:"vendor,omitempty"`
	Type             string     `json:"type,omitempty"`
	IntegrationKey   string     `json:"integration_key,omitempty"`
	IntegrationEmail string     `json:"integration_email,omitempty"`
}

// InlineModel represents when a scheduled action will occur.
type InlineModel struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// ScheduledAction contains scheduled actions for the service.
type ScheduledAction struct {
	Type      string      `json:"type,omitempty"`
	At        InlineModel `json:"at,omitempty"`
	ToUrgency string      `json:"to_urgency"`
}

// IncidentUrgencyType are the incidents urgency during or outside support hours.
type IncidentUrgencyType struct {
	Type    string `json:"type,omitempty"`
	Urgency string `json:"urgency,omitempty"`
}

// SupportHours are the support hours for the service.
type SupportHours struct {
	Type       string `json:"type,omitempty"`
	Timezone   string `json:"time_zone,omitempty"`
	StartTime  string `json:"start_time,omitempty"`
	EndTime    string `json:"end_time,omitempty"`
	DaysOfWeek []uint `json:"days_of_week,omitempty"`
}

// IncidentUrgencyRule is the default urgency for new incidents.
type IncidentUrgencyRule struct {
	Type                string               `json:"type,omitempty"`
	Urgency             string               `json:"urgency,omitempty"`
	DuringSupportHours  *IncidentUrgencyType `json:"during_support_hours,omitempty"`
	OutsideSupportHours *IncidentUrgencyType `json:"outside_support_hours,omitempty"`
}

// Service represents something you monitor (like a web service, email service, or database service).
type Service struct {
	APIObject
	Name                   string               `json:"name,omitempty"`
	Description            string               `json:"description,omitempty"`
	AutoResolveTimeout     *uint                `json:"auto_resolve_timeout"`
	AcknowledgementTimeout *uint                `json:"acknowledgement_timeout"`
	CreateAt               string               `json:"created_at,omitempty"`
	Status                 string               `json:"status,omitempty"`
	LastIncidentTimestamp  string               `json:"last_incident_timestamp,omitempty"`
	Integrations           []Integration        `json:"integrations,omitempty"`
	EscalationPolicy       EscalationPolicy     `json:"escalation_policy,omitempty"`
	Teams                  []Team               `json:"teams,omitempty"`
	IncidentUrgencyRule    *IncidentUrgencyRule `json:"incident_urgency_rule,omitempty"`
	SupportHours           *SupportHours        `json:"support_hours,omitempty"`
	ScheduledActions       []ScheduledAction    `json:"scheduled_actions,omitempty"`
}

// ListServiceOptions is the data structure used when calling the ListServices API endpoint.
type ListServiceOptions struct {
	APIListObject
	TeamIDs  []string `url:"team_ids,omitempty,brackets"`
	TimeZone string   `url:"time_zone,omitempty"`
	SortBy   string   `url:"sort_by,omitempty"`
	Query    string   `url:"query,omitempty"`
	Includes []string `url:"include,omitempty,brackets"`
}

// ListServiceResponse is the data structure returned from calling the ListServices API endpoint.
type ListServiceResponse struct {
	APIListObject
	Services []Service
}

// ListServices lists existing services.
func (c *Client) ListServices(o ListServiceOptions) (*ListServiceResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}
	resp, err := c.get("/services?" + v.Encode())
	if err != nil {
		return nil, err
	}
	var result ListServiceResponse
	return &result, c.decodeJSON(resp, &result)
}

// GetServiceOptions is the data structure used when calling the GetService API endpoint.
type GetServiceOptions struct {
	Includes []string `url:"include,brackets,omitempty"`
}

// GetService gets details about an existing service.
func (c *Client) GetService(id string, o *GetServiceOptions) (*Service, error) {
	v, err := query.Values(o)
	resp, err := c.get("/services/" + id + "?" + v.Encode())
	return getServiceFromResponse(c, resp, err)
}

// CreateService creates a new service.
func (c *Client) CreateService(s Service) (*Service, error) {
	data := make(map[string]Service)
	data["service"] = s
	resp, err := c.post("/services", data)
	return getServiceFromResponse(c, resp, err)
}

// UpdateService updates an existing service.
func (c *Client) UpdateService(s Service) (*Service, error) {
	resp, err := c.put("/services/"+s.ID, s, nil)
	return getServiceFromResponse(c, resp, err)
}

// DeleteService deletes an existing service.
func (c *Client) DeleteService(id string) error {
	_, err := c.delete("/services/" + id)
	return err
}

// CreateIntegration creates a new integration belonging to a service.
func (c *Client) CreateIntegration(id string, i Integration) (*Integration, error) {
	data := make(map[string]Integration)
	data["integration"] = i
	resp, err := c.post("/services/"+id+"/integrations", data)
	return getIntegrationFromResponse(c, resp, err)
}

// GetIntegrationOptions is the data structure used when calling the GetIntegration API endpoint.
type GetIntegrationOptions struct {
	Includes []string `url:"include,omitempty,brackets"`
}

// GetIntegration gets details about an integration belonging to a service.
func (c *Client) GetIntegration(serviceID, integrationID string, o GetIntegrationOptions) (*Integration, error) {
	v, queryErr := query.Values(o)
	if queryErr != nil {
		return nil, queryErr
	}
	resp, err := c.get("/services/" + serviceID + "/integrations/" + integrationID + "?" + v.Encode())
	return getIntegrationFromResponse(c, resp, err)
}

// UpdateIntegration updates an integration belonging to a service.
func (c *Client) UpdateIntegration(serviceID string, i Integration) (*Integration, error) {
	resp, err := c.put("/services/"+serviceID+"/integrations/"+i.ID, i, nil)
	return getIntegrationFromResponse(c, resp, err)
}

// DeleteIntegration deletes an existing integration.
func (c *Client) DeleteIntegration(serviceID string, integrationID string) error {
	_, err := c.delete("/services/" + serviceID + "/integrations/" + integrationID)
	return err
}

func getServiceFromResponse(c *Client, resp *http.Response, err error) (*Service, error) {
	if err != nil {
		return nil, err
	}
	var target map[string]Service
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}
	rootNode := "service"
	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}
	return &t, nil
}

func getIntegrationFromResponse(c *Client, resp *http.Response, err error) (*Integration, error) {
	if err != nil {
		return nil, err
	}
	var target map[string]Integration
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", err)
	}
	rootNode := "integration"
	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}
	return &t, nil
}
