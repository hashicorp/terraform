package api

import (
	"fmt"
	"net/url"
)

func (c *Client) queryComponentMetricData(componentID int, names []string) ([]Metric, error) {
	data := []Metric{}

	reqURL, err := url.Parse(fmt.Sprintf("/components/%v/metrics/data.json", componentID))
	if err != nil {
		return nil, err
	}

	qs := reqURL.Query()
	for _, name := range names {
		qs.Add("names[]", name)
	}
	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			MetricData struct {
				Metrics []Metric `json:"metrics"`
			} `json:"metric_data,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		data = append(data, resp.MetricData.Metrics...)
	}

	return data, nil
}

// ListComponentMetricData lists all the metric data for the specified component ID and metric names.
func (c *Client) ListComponentMetricData(componentID int, names []string) ([]Metric, error) {
	return c.queryComponentMetricData(componentID, names)
}
