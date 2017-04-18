package api

import (
	"fmt"
	"net/url"
)

func (c *Client) queryLabels() ([]Label, error) {
	labels := []Label{}

	reqURL, err := url.Parse("/labels.json")
	if err != nil {
		return nil, err
	}

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			Labels []Label `json:"labels,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		labels = append(labels, resp.Labels...)
	}

	return labels, nil
}

// GetLabel gets the label for the specified key.
func (c *Client) GetLabel(key string) (*Label, error) {
	labels, err := c.queryLabels()
	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		if label.Key == key {
			return &label, nil
		}
	}

	return nil, ErrNotFound
}

// ListLabels returns the labels for the account.
func (c *Client) ListLabels() ([]Label, error) {
	return c.queryLabels()
}

// CreateLabel creates a new label for the account.
func (c *Client) CreateLabel(label Label) error {
	if label.Links.Applications == nil {
		label.Links.Applications = make([]int, 0)
	}

	if label.Links.Servers == nil {
		label.Links.Servers = make([]int, 0)
	}

	req := struct {
		Label Label `json:"label,omitempty"`
	}{
		Label: label,
	}

	_, err := c.Do("PUT", "/labels.json", req, nil)
	return err
}

// DeleteLabel deletes a label on the account specified by key.
func (c *Client) DeleteLabel(key string) error {
	u := &url.URL{Path: fmt.Sprintf("/labels/%v.json", key)}
	_, err := c.Do("DELETE", u.String(), nil, nil)
	return err
}
