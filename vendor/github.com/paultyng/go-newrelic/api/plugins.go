package api

import (
	"net/url"
	"strconv"
)

type pluginsFilters struct {
	GUID *string
	IDs  []int
}

func (c *Client) queryPlugins(filters pluginsFilters) ([]Plugin, error) {
	plugins := []Plugin{}

	reqURL, err := url.Parse("/plugins.json")
	if err != nil {
		return nil, err
	}

	qs := reqURL.Query()
	if filters.GUID != nil {
		qs.Set("filter[guid]", *filters.GUID)
	}
	for _, id := range filters.IDs {
		qs.Add("filter[ids]", strconv.Itoa(id))
	}
	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			Plugins []Plugin `json:"plugins,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		plugins = append(plugins, resp.Plugins...)
	}

	return plugins, nil
}

// ListPlugins lists all the plugins you have access to.
func (c *Client) ListPlugins() ([]Plugin, error) {
	return c.queryPlugins(pluginsFilters{})
}
