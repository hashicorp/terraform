package api

import (
	"fmt"
	"net/url"
	"strconv"
)

func (c *Client) queryAlertConditions(policyID int) ([]AlertCondition, error) {
	conditions := []AlertCondition{}

	reqURL, err := url.Parse("/alerts_conditions.json")
	if err != nil {
		return nil, err
	}

	qs := reqURL.Query()
	qs.Set("policy_id", strconv.Itoa(policyID))

	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			Conditions []AlertCondition `json:"conditions,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		for _, c := range resp.Conditions {
			c.PolicyID = policyID
		}

		conditions = append(conditions, resp.Conditions...)
	}

	return conditions, nil
}

// GetAlertCondition gets information about an alert condition given an ID and policy ID.
func (c *Client) GetAlertCondition(policyID int, id int) (*AlertCondition, error) {
	conditions, err := c.queryAlertConditions(policyID)
	if err != nil {
		return nil, err
	}

	for _, condition := range conditions {
		if condition.ID == id {
			return &condition, nil
		}
	}

	return nil, ErrNotFound
}

// ListAlertConditions returns alert conditions for the specified policy.
func (c *Client) ListAlertConditions(policyID int) ([]AlertCondition, error) {
	return c.queryAlertConditions(policyID)
}

// CreateAlertCondition creates an alert condition given the passed configuration.
func (c *Client) CreateAlertCondition(condition AlertCondition) (*AlertCondition, error) {
	policyID := condition.PolicyID

	req := struct {
		Condition AlertCondition `json:"condition"`
	}{
		Condition: condition,
	}

	resp := struct {
		Condition AlertCondition `json:"condition,omitempty"`
	}{}

	u := &url.URL{Path: fmt.Sprintf("/alerts_conditions/policies/%v.json", policyID)}
	_, err := c.Do("POST", u.String(), req, &resp)
	if err != nil {
		return nil, err
	}

	resp.Condition.PolicyID = policyID

	return &resp.Condition, nil
}

// UpdateAlertCondition updates an alert condition with the specified changes.
func (c *Client) UpdateAlertCondition(condition AlertCondition) (*AlertCondition, error) {
	policyID := condition.PolicyID
	id := condition.ID

	req := struct {
		Condition AlertCondition `json:"condition"`
	}{
		Condition: condition,
	}

	resp := struct {
		Condition AlertCondition `json:"condition,omitempty"`
	}{}

	u := &url.URL{Path: fmt.Sprintf("/alerts_conditions/%v.json", id)}
	_, err := c.Do("PUT", u.String(), req, &resp)
	if err != nil {
		return nil, err
	}

	resp.Condition.PolicyID = policyID

	return &resp.Condition, nil
}

// DeleteAlertCondition removes the alert condition given the specified ID and policy ID.
func (c *Client) DeleteAlertCondition(policyID int, id int) error {
	u := &url.URL{Path: fmt.Sprintf("/alerts_conditions/%v.json", id)}
	_, err := c.Do("DELETE", u.String(), nil, nil)
	return err
}
