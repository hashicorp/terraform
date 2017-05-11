package api

import (
	"net/url"
	"strconv"
)

type applicationsFilters struct {
	Name     *string
	Host     *string
	IDs      []int
	Language *string
}

func (c *Client) queryApplications(filters applicationsFilters) ([]Application, error) {
	applications := []Application{}

	reqURL, err := url.Parse("/applications.json")
	if err != nil {
		return nil, err
	}

	qs := reqURL.Query()
	if filters.Name != nil {
		qs.Set("filter[name]", *filters.Name)
	}
	if filters.Host != nil {
		qs.Set("filter[host]", *filters.Host)
	}
	for _, id := range filters.IDs {
		qs.Add("filter[ids]", strconv.Itoa(id))
	}
	if filters.Language != nil {
		qs.Set("filter[language]", *filters.Language)
	}
	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			Applications []Application `json:"applications,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		applications = append(applications, resp.Applications...)
	}

	return applications, nil
}

// ListApplications lists all the applications you have access to.
func (c *Client) ListApplications() ([]Application, error) {
	return c.queryApplications(applicationsFilters{})
}
