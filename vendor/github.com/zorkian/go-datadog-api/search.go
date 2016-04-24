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
func (self *Client) SearchHosts(search string) ([]string, error) {
	var out reqSearch
	err := self.doJsonRequest("GET", "/v1/search?q=hosts:"+search, nil, &out)
	if err != nil {
		return nil, err
	}
	return out.Results.Hosts, nil
}

// SearchMetrics searches through the metrics facet, returning matching ones.
func (self *Client) SearchMetrics(search string) ([]string, error) {
	var out reqSearch
	err := self.doJsonRequest("GET", "/v1/search?q=metrics:"+search, nil, &out)
	if err != nil {
		return nil, err
	}
	return out.Results.Metrics, nil
}
