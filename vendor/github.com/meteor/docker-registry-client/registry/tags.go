package registry

type tagsResponse struct {
	Tags []string `json:"tags"`
}

func (registry *Registry) Tags(repository string) ([]string, error) {
	url := registry.url("/v2/%s/tags/list", repository)
	registry.Logf("registry.tags url=%s repository=%s", url, repository)

	var response tagsResponse
	if err := registry.getJson(url, &response); err != nil {
		return nil, err
	}

	return response.Tags, nil
}
