package api

import (
	"net/url"
	"strconv"
)

type componentsFilters struct {
	PluginID int
	IDs      []int
}

func (c *Client) queryComponents(filters componentsFilters) ([]Component, error) {
	components := []Component{}

	reqURL, err := url.Parse("/components.json")
	if err != nil {
		return nil, err
	}

	qs := reqURL.Query()
	qs.Set("filter[plugin_id]", strconv.Itoa(filters.PluginID))
	for _, id := range filters.IDs {
		qs.Add("filter[ids]", strconv.Itoa(id))
	}
	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			Components []Component `json:"components,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		components = append(components, resp.Components...)
	}

	return components, nil
}

// ListComponents lists all the components for the specified plugin ID.
func (c *Client) ListComponents(pluginID int) ([]Component, error) {
	return c.queryComponents(componentsFilters{
		PluginID: pluginID,
	})
}
