package pagerduty

import (
	"fmt"

	"github.com/google/go-querystring/query"
)

// Agent is the actor who carried out the action.
type Agent APIObject

// Channel is the means by which the action was carried out.
type Channel struct {
	Type string
}

// Context are to be included with the trigger such as links to graphs or images.
type Context struct {
	Alt  string
	Href string
	Src  string
	Text string
	Type string
}

// LogEntry is a list of all of the events that happened to an incident.
type LogEntry struct {
	APIObject
	CreatedAt              string `json:"created_at"`
	Agent                  Agent
	Channel                Channel
	Incident               Incident
	Teams                  []Team
	Contexts               []Context
	AcknowledgementTimeout int `json:"acknowledgement_timeout"`
	EventDetails           map[string]string
}

// ListLogEntryResponse is the response data when calling the ListLogEntry API endpoint.
type ListLogEntryResponse struct {
	APIListObject
	LogEntries []LogEntry `json:"log_entries"`
}

// ListLogEntriesOptions is the data structure used when calling the ListLogEntry API endpoint.
type ListLogEntriesOptions struct {
	APIListObject
	TimeZone   string   `url:"time_zone"`
	Since      string   `url:"since,omitempty"`
	Until      string   `url:"until,omitempty"`
	IsOverview bool     `url:"is_overview,omitempty"`
	Includes   []string `url:"include,omitempty,brackets"`
}

// ListLogEntries lists all of the incident log entries across the entire account.
func (c *Client) ListLogEntries(o ListLogEntriesOptions) (*ListLogEntryResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}
	resp, err := c.get("/log_entries?" + v.Encode())
	if err != nil {
		return nil, err
	}
	var result ListLogEntryResponse
	return &result, c.decodeJSON(resp, &result)
}

// GetLogEntryOptions is the data structure used when calling the GetLogEntry API endpoint.
type GetLogEntryOptions struct {
	TimeZone string   `url:"timezone,omitempty"`
	Includes []string `url:"include,omitempty,brackets"`
}

// GetLogEntry list log entries for the specified incident.
func (c *Client) GetLogEntry(id string, o GetLogEntryOptions) (*LogEntry, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}
	resp, err := c.get("/log_entries/" + id + "?" + v.Encode())
	if err != nil {
		return nil, err
	}
	var result map[string]LogEntry
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}
	le, ok := result["log_entry"]
	if !ok {
		return nil, fmt.Errorf("JSON response does not have log_entry field")
	}
	return &le, nil
}
