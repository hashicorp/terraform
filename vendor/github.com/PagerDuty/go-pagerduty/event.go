package pagerduty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const eventEndPoint = "https://events.pagerduty.com/generic/2010-04-15/create_event.json"

// Event stores data for problem reporting, acknowledgement, and resolution.
type Event struct {
	ServiceKey  string        `json:"service_key"`
	Type        string        `json:"event_type"`
	IncidentKey string        `json:"incident_key,omitempty"`
	Description string        `json:"description"`
	Client      string        `json:"client,omitempty"`
	ClientURL   string        `json:"client_url,omitempty"`
	Details     interface{}   `json:"details,omitempty"`
	Contexts    []interface{} `json:"contexts,omitempty"`
}

// EventResponse is the data returned from the CreateEvent API endpoint.
type EventResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	IncidentKey string `json:"incident_key"`
}

// CreateEvent sends PagerDuty an event to report, acknowledge, or resolve a problem.
func CreateEvent(e Event) (*EventResponse, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("POST", eventEndPoint, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Status Code: %d", resp.StatusCode)
	}
	var eventResponse EventResponse
	if err := json.NewDecoder(resp.Body).Decode(&eventResponse); err != nil {
		return nil, err
	}
	return &eventResponse, nil
}
