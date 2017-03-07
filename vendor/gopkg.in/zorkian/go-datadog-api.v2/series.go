/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

import "strconv"

// DataPoint is a tuple of [UNIX timestamp, value]. This has to use floats
// because the value could be non-integer.
type DataPoint [2]float64

// Metric represents a collection of data points that we might send or receive
// on one single metric line.
type Metric struct {
	Metric *string     `json:"metric,omitempty"`
	Points []DataPoint `json:"points,omitempty"`
	Type   *string     `json:"type,omitempty"`
	Host   *string     `json:"host,omitempty"`
	Tags   []string    `json:"tags,omitempty"`
	Unit   *string     `json:"unit,omitempty"`
}

// Series represents a collection of data points we get when we query for timeseries data
type Series struct {
	Metric      *string     `json:"metric,omitempty"`
	DisplayName *string     `json:"display_name,omitempty"`
	Points      []DataPoint `json:"pointlist,omitempty"`
	Start       *float64    `json:"start,omitempty"`
	End         *float64    `json:"end,omitempty"`
	Interval    *int        `json:"interval,omitempty"`
	Aggr        *string     `json:"aggr,omitempty"`
	Length      *int        `json:"length,omitempty"`
	Scope       *string     `json:"scope,omitempty"`
	Expression  *string     `json:"expression,omitempty"`
}

// reqPostSeries from /api/v1/series
type reqPostSeries struct {
	Series []Metric `json:"series,omitempty"`
}

// reqMetrics is the container for receiving metric results.
type reqMetrics struct {
	Series []Series `json:"series,omitempty"`
}

// PostMetrics takes as input a slice of metrics and then posts them up to the
// server for posting data.
func (client *Client) PostMetrics(series []Metric) error {
	return client.doJsonRequest("POST", "/v1/series",
		reqPostSeries{Series: series}, nil)
}

// QueryMetrics takes as input from, to (seconds from Unix Epoch) and query string and then requests
// timeseries data for that time peried
func (client *Client) QueryMetrics(from, to int64, query string) ([]Series, error) {
	var out reqMetrics
	if err := client.doJsonRequest("GET", "/v1/query?from="+strconv.FormatInt(from, 10)+"&to="+strconv.FormatInt(to, 10)+"&query="+query,
		nil, &out); err != nil {
		return nil, err
	}
	return out.Series, nil
}
