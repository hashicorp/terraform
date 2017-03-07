/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

// TagMap is used to receive the format given to us by the API.
type TagMap map[string][]string

// reqGetTags is the container for receiving tags.
type reqGetTags struct {
	Tags *TagMap `json:"tags,omitempty"`
}

// regGetHostTags is for receiving a slice of tags.
type reqGetHostTags struct {
	Tags []string `json:"tags,omitempty"`
}

// GetTags returns a map of tags.
func (client *Client) GetTags(source string) (TagMap, error) {
	var out reqGetTags
	uri := "/v1/tags/hosts"
	if source != "" {
		uri += "?source=" + source
	}
	if err := client.doJsonRequest("GET", uri, nil, &out); err != nil {
		return nil, err
	}
	return *out.Tags, nil
}

// GetHostTags returns a slice of tags for a given host and source.
func (client *Client) GetHostTags(host, source string) ([]string, error) {
	var out reqGetHostTags
	uri := "/v1/tags/hosts/" + host
	if source != "" {
		uri += "?source=" + source
	}
	if err := client.doJsonRequest("GET", uri, nil, &out); err != nil {
		return nil, err
	}
	return out.Tags, nil
}

// GetHostTagsBySource is a different way of viewing the tags. It returns a map
// of source:[tag,tag].
func (client *Client) GetHostTagsBySource(host, source string) (TagMap, error) {
	var out reqGetTags
	uri := "/v1/tags/hosts/" + host + "?by_source=true"
	if source != "" {
		uri += "&source=" + source
	}
	if err := client.doJsonRequest("GET", uri, nil, &out); err != nil {
		return nil, err
	}
	return *out.Tags, nil
}

// AddTagsToHost does exactly what it says on the tin. Given a list of tags,
// add them to the host. The source is optionally specified, and defaults to
// "users" as per the API documentation.
func (client *Client) AddTagsToHost(host, source string, tags []string) error {
	uri := "/v1/tags/hosts/" + host
	if source != "" {
		uri += "?source=" + source
	}
	return client.doJsonRequest("POST", uri, reqGetHostTags{Tags: tags}, nil)
}

// UpdateHostTags overwrites existing tags for a host, allowing you to specify
// a new set of tags for the given source. This defaults to "users".
func (client *Client) UpdateHostTags(host, source string, tags []string) error {
	uri := "/v1/tags/hosts/" + host
	if source != "" {
		uri += "?source=" + source
	}
	return client.doJsonRequest("PUT", uri, reqGetHostTags{Tags: tags}, nil)
}

// RemoveHostTags removes all tags from a host for the given source. If none is
// given, the API defaults to "users".
func (client *Client) RemoveHostTags(host, source string) error {
	uri := "/v1/tags/hosts/" + host
	if source != "" {
		uri += "?source=" + source
	}
	return client.doJsonRequest("DELETE", uri, nil, nil)
}
