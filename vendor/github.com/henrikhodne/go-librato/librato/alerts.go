package librato

import (
	"fmt"
	"net/http"
)

// AlertsService handles communication with the Librato API methods related to
// alerts.
type AlertsService struct {
	client *Client
}

// Alert represents a Librato Alert.
type Alert struct {
	Name       *string          `json:"name"`
	ID         *uint            `json:"id,omitempty"`
	Conditions []AlertCondition `json:"conditions,omitempty"`
	// These are interface{} because the Librato API asks for integers
	// on Create and returns hashes on Get
	Services     interface{}      `json:"services,omitempty"`
	Attributes   *AlertAttributes `json:"attributes,omitempty"`
	Description  *string          `json:"description,omitempty"`
	Active       *bool            `json:"active,omitempty"`
	RearmSeconds *uint            `json:"rearm_seconds,omitempty"`
}

func (a Alert) String() string {
	return Stringify(a)
}

// AlertCondition represents an alert trigger condition.
type AlertCondition struct {
	Type            *string  `json:"type,omitempty"`
	MetricName      *string  `json:"metric_name,omitempty"`
	Source          *string  `json:"source,omitempty"`
	DetectReset     *bool    `json:"detect_reset,omitempty"`
	Threshold       *float64 `json:"threshold,omitempty"`
	SummaryFunction *string  `json:"summary_function,omitempty"`
	Duration        *uint    `json:"duration,omitempty"`
}

// AlertAttributes represents the attributes of an alert.
type AlertAttributes struct {
	RunbookURL *string `json:"runbook_url,omitempty"`
}

// Get an alert by ID
//
// Librato API docs: https://www.librato.com/docs/api/#retrieve-alert-by-id
func (a *AlertsService) Get(id uint) (*Alert, *http.Response, error) {
	urlStr := fmt.Sprintf("alerts/%d", id)

	req, err := a.client.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, nil, err
	}

	alert := new(Alert)
	resp, err := a.client.Do(req, alert)
	if err != nil {
		return nil, resp, err
	}

	return alert, resp, err
}

// Create an alert
//
// Librato API docs: https://www.librato.com/docs/api/?shell#create-an-alert
func (a *AlertsService) Create(alert *Alert) (*Alert, *http.Response, error) {
	req, err := a.client.NewRequest("POST", "alerts", alert)
	if err != nil {
		return nil, nil, err
	}

	al := new(Alert)
	resp, err := a.client.Do(req, al)
	if err != nil {
		return nil, resp, err
	}

	return al, resp, err
}

// Update an alert.
//
// Librato API docs: https://www.librato.com/docs/api/?shell#update-alert
func (a *AlertsService) Update(alertID uint, alert *Alert) (*http.Response, error) {
	u := fmt.Sprintf("alerts/%d", alertID)
	req, err := a.client.NewRequest("PUT", u, alert)
	if err != nil {
		return nil, err
	}

	return a.client.Do(req, nil)
}

// Delete an alert
//
// Librato API docs: https://www.librato.com/docs/api/?shell#delete-alert
func (a *AlertsService) Delete(id uint) (*http.Response, error) {
	u := fmt.Sprintf("alerts/%d", id)
	req, err := a.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return a.client.Do(req, nil)
}
