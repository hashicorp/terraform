package pagerduty

import (
	"fmt"
	"github.com/google/go-querystring/query"
	"net/http"
)

const (
	escPath = "/escalation_policies"
)

// EscalationRule is a rule for an escalation policy to trigger.
type EscalationRule struct {
	ID      string      `json:"id,omitempty"`
	Delay   uint        `json:"escalation_delay_in_minutes,omitempty"`
	Targets []APIObject `json:"targets"`
}

// EscalationPolicy is a collection of escalation rules.
type EscalationPolicy struct {
	APIObject
	Name            string           `json:"name,omitempty"`
	EscalationRules []EscalationRule `json:"escalation_rules,omitempty"`
	Services        []APIReference   `json:"services,omitempty"`
	NumLoops        uint             `json:"num_loops,omitempty"`
	Teams           []APIReference   `json:"teams,omitempty"`
	Description     string           `json:"description,omitempty"`
	RepeatEnabled   bool             `json:"repeat_enabled,omitempty"`
}

// ListEscalationPoliciesResponse is the data structure returned from calling the ListEscalationPolicies API endpoint.
type ListEscalationPoliciesResponse struct {
	APIListObject
	EscalationPolicies []EscalationPolicy `json:"escalation_policies"`
}

// ListEscalationPoliciesOptions is the data structure used when calling the ListEscalationPolicies API endpoint.
type ListEscalationPoliciesOptions struct {
	APIListObject
	Query    string   `url:"query,omitempty"`
	UserIDs  []string `url:"user_ids,omitempty,brackets"`
	TeamIDs  []string `url:"team_ids,omitempty,brackets"`
	Includes []string `url:"include,omitempty,brackets"`
	SortBy   string   `url:"sort_by,omitempty"`
}

// ListEscalationPolicies lists all of the existing escalation policies.
func (c *Client) ListEscalationPolicies(o ListEscalationPoliciesOptions) (*ListEscalationPoliciesResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}
	resp, err := c.get(escPath + "?" + v.Encode())
	if err != nil {
		return nil, err
	}
	var result ListEscalationPoliciesResponse
	return &result, c.decodeJSON(resp, &result)
}

// CreateEscalationPolicy creates a new escalation policy.
func (c *Client) CreateEscalationPolicy(e EscalationPolicy) (*EscalationPolicy, error) {
	data := make(map[string]EscalationPolicy)
	data["escalation_policy"] = e
	resp, err := c.post(escPath, data)
	return getEscalationPolicyFromResponse(c, resp, err)
}

// DeleteEscalationPolicy deletes an existing escalation policy and rules.
func (c *Client) DeleteEscalationPolicy(id string) error {
	_, err := c.delete(escPath + "/" + id)
	return err
}

// GetEscalationPolicyOptions is the data structure used when calling the GetEscalationPolicy API endpoint.
type GetEscalationPolicyOptions struct {
	Includes []string `url:"include,omitempty,brackets"`
}

// GetEscalationPolicy gets information about an existing escalation policy and its rules.
func (c *Client) GetEscalationPolicy(id string, o *GetEscalationPolicyOptions) (*EscalationPolicy, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}
	resp, err := c.get(escPath + "/" + id + "?" + v.Encode())
	return getEscalationPolicyFromResponse(c, resp, err)
}

// UpdateEscalationPolicy updates an existing escalation policy and its rules.
func (c *Client) UpdateEscalationPolicy(id string, e *EscalationPolicy) (*EscalationPolicy, error) {
	data := make(map[string]EscalationPolicy)
	data["escalation_policy"] = *e
	resp, err := c.put(escPath+"/"+id, data, nil)
	return getEscalationPolicyFromResponse(c, resp, err)
}

func getEscalationPolicyFromResponse(c *Client, resp *http.Response, err error) (*EscalationPolicy, error) {
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var target map[string]EscalationPolicy
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}
	rootNode := "escalation_policy"
	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}
	return &t, nil
}
