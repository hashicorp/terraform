package client

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudfoundry/go-ccapi/v3/routing"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter -o fakes/fake_client.go . Client
type Client interface {
	GetApplications(queryParams url.Values) ([]byte, error)
	GetResource(path string) ([]byte, error)
	GetResources(path string, limit int) ([]byte, error)

	TokensUpdated() bool
	GetUpdatedTokens() (string, string)
}

type client struct {
	ccEndpoint   string
	accessToken  string
	refreshToken string

	ccRequestGenerator *rata.RequestGenerator
	httpClient         *http.Client
	baseFetcher        Fetcher

	updatedAccessToken  string
	updatedRefreshToken string
}

func NewClient(ccEndpoint, uaaEndpoint, accessToken, refreshToken string) Client {
	tokenRefresher := NewTokenRefresher(uaaEndpoint)
	baseFetcher := NewBaseFetcher(tokenRefresher, refreshToken)
	return &client{
		ccEndpoint:   ccEndpoint,
		accessToken:  accessToken,
		refreshToken: refreshToken,

		ccRequestGenerator: rata.NewRequestGenerator(ccEndpoint, routing.CCRoutes),
		httpClient:         &http.Client{},
		baseFetcher:        baseFetcher,
	}
}

func (c client) TokensUpdated() bool {
	return c.updatedAccessToken != "" && c.updatedAccessToken != c.accessToken
}

func (c client) GetUpdatedTokens() (string, string) {
	return c.updatedAccessToken, c.updatedRefreshToken
}

func (c client) GetApplications(queryParams url.Values) ([]byte, error) {
	req, err := c.ccRequestGenerator.CreateRequest("apps", rata.Params{}, strings.NewReader(""))
	if err != nil {
		return []byte{}, err
	}

	req.URL.RawQuery = queryParams.Encode()
	req.Header.Set("Authorization", c.accessToken)

	paginatedResourceFetcher := NewPaginatedResourceFetcher(0, c.baseFetcher, c.refreshToken)
	responseJSON, err := paginatedResourceFetcher.Fetch(req)
	if err != nil {
		return []byte{}, err
	}

	c.updatedAccessToken, c.updatedRefreshToken = paginatedResourceFetcher.GetUpdatedTokens()

	return responseJSON, nil
}

func (c client) GetResource(path string) ([]byte, error) {
	url := c.ccEndpoint + "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Set("Authorization", c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return responseBytes, nil
}

func (c client) GetResources(path string, limit int) ([]byte, error) {
	u, err := url.Parse(path)
	if err != nil {
		return []byte{}, err
	}

	req, err := http.NewRequest("GET", c.ccEndpoint+u.Path, nil)
	if err != nil {
		return []byte{}, err
	}

	req.URL.RawQuery = u.Query().Encode()
	req.Header.Set("Authorization", c.accessToken)

	paginatedResourceFetcher := NewPaginatedResourceFetcher(limit, c.baseFetcher, c.refreshToken)

	return paginatedResourceFetcher.Fetch(req)
}
