package api

import (
	"fmt"
	"net/url"
)

func (c *Client) queryAlertPolicies(name *string) ([]AlertPolicy, error) {
	policies := []AlertPolicy{}

	reqURL, err := url.Parse("/alerts_policies.json")
	if err != nil {
		return nil, err
	}

	qs := reqURL.Query()
	if name != nil {
		qs.Set("filter[name]", *name)
	}
	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			Policies []AlertPolicy `json:"policies,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		policies = append(policies, resp.Policies...)
	}

	return policies, nil
}

// GetAlertPolicy returns a specific alert policy by ID
func (c *Client) GetAlertPolicy(id int) (*AlertPolicy, error) {
	policies, err := c.queryAlertPolicies(nil)
	if err != nil {
		return nil, err
	}

	for _, policy := range policies {
		if policy.ID == id {
			return &policy, nil
		}
	}

	return nil, ErrNotFound
}

// ListAlertPolicies returns all alert policies for the account.
func (c *Client) ListAlertPolicies() ([]AlertPolicy, error) {
	return c.queryAlertPolicies(nil)
}

// CreateAlertPolicy creates a new alert policy for the account.
func (c *Client) CreateAlertPolicy(policy AlertPolicy) (*AlertPolicy, error) {
	req := struct {
		Policy AlertPolicy `json:"policy"`
	}{
		Policy: policy,
	}

	resp := struct {
		Policy AlertPolicy `json:"policy,omitempty"`
	}{}

	_, err := c.Do("POST", "/alerts_policies.json", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Policy, nil
}

// DeleteAlertPolicy deletes an existing alert policy from the account.
func (c *Client) DeleteAlertPolicy(id int) error {
	u := &url.URL{Path: fmt.Sprintf("/alerts_policies/%v.json", id)}
	_, err := c.Do("DELETE", u.String(), nil, nil)
	return err
}
