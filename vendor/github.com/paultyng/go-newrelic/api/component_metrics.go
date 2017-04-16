package api

import (
	"fmt"
	"net/url"
)

func (c *Client) queryComponentMetrics(componentID int) ([]ComponentMetric, error) {
	metrics := []ComponentMetric{}

	reqURL, err := url.Parse(fmt.Sprintf("/components/%v/metrics.json", componentID))
	if err != nil {
		return nil, err
	}

	qs := reqURL.Query()
	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			Metrics []ComponentMetric `json:"metrics,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		metrics = append(metrics, resp.Metrics...)
	}

	return metrics, nil
}

// ListComponentMetrics lists all the component metrics for the specificed component ID.
func (c *Client) ListComponentMetrics(componentID int) ([]ComponentMetric, error) {
	return c.queryComponentMetrics(componentID)
}
