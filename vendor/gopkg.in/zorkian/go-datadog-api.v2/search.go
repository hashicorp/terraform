/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

// reqSearch is the container for receiving search results.
type reqSearch struct {
	Results struct {
		Hosts   []string `json:"hosts,omitempty"`
		Metrics []string `json:"metrics,omitempty"`
	} `json:"results"`
}

// SearchHosts searches through the hosts facet, returning matching hostnames.
func (client *Client) SearchHosts(search string) ([]string, error) {
	var out reqSearch
	if err := client.doJsonRequest("GET", "/v1/search?q=hosts:"+search, nil, &out); err != nil {
		return nil, err
	}
	return out.Results.Hosts, nil
}

// SearchMetrics searches through the metrics facet, returning matching ones.
func (client *Client) SearchMetrics(search string) ([]string, error) {
	var out reqSearch
	if err := client.doJsonRequest("GET", "/v1/search?q=metrics:"+search, nil, &out); err != nil {
		return nil, err
	}
	return out.Results.Metrics, nil
}
