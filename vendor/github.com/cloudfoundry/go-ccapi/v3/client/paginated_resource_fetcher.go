package client

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type paginatedResourceFetcher struct {
	limit        int
	baseFetcher  Fetcher
	refreshToken string
}

func NewPaginatedResourceFetcher(
	limit int,
	baseFetcher Fetcher,
	refreshToken string,
) Fetcher {
	return &paginatedResourceFetcher{
		baseFetcher:  baseFetcher,
		limit:        limit,
		refreshToken: refreshToken,
	}
}

func (f paginatedResourceFetcher) GetUpdatedTokens() (string, string) {
	return f.baseFetcher.GetUpdatedTokens()
}

func (f paginatedResourceFetcher) Fetch(req *http.Request) ([]byte, error) {
	resources, nextPath, err := f.performRequest(req)
	if err != nil {
		return []byte{}, err
	}

	var rs []interface{}
	var nextReq *http.Request

	for nextPath != nil && (f.limit == 0 || len(resources) < f.limit) {
		u, err := url.Parse(*nextPath)
		if err != nil {
			return []byte{}, err
		}

		nextReq = &http.Request{
			URL: &url.URL{
				Scheme:   req.URL.Scheme,
				Host:     req.URL.Host,
				Path:     u.Path,
				RawQuery: u.Query().Encode(),
			},
		}

		rs, nextPath, err = f.performRequest(nextReq)
		if err != nil {
			return []byte{}, err
		}

		resources = append(resources, rs...)
	}

	if f.limit > 0 {
		resources = resources[:f.limit]
	}

	responseJSON, err := json.Marshal(resources)
	if err != nil {
		return []byte{}, err
	}

	return responseJSON, nil
}

func (f paginatedResourceFetcher) performRequest(req *http.Request) ([]interface{}, *string, error) {
	responseBytes, err := f.baseFetcher.Fetch(req)
	if err != nil {
		return []interface{}{}, nil, err
	}

	response := &GetResourcesResponse{}
	err = json.Unmarshal(responseBytes, response)
	if err != nil {
		return []interface{}{}, nil, err
	}

	return response.Resources, response.Pagination.Next, nil
}
