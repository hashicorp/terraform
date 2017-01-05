package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const invalidTokenCode = 1000

//go:generate counterfeiter -o fakes/fake_fetcher.go . Fetcher
type Fetcher interface {
	Fetch(req *http.Request) ([]byte, error)
	GetUpdatedTokens() (string, string)
}

type baseFetcher struct {
	httpClient     *http.Client
	tokenRefresher TokenRefresher
	accessToken    string
	refreshToken   string
}

func NewBaseFetcher(tokenRefresher TokenRefresher, refreshToken string) Fetcher {
	return &baseFetcher{
		httpClient:     &http.Client{},
		tokenRefresher: tokenRefresher,
		refreshToken:   refreshToken,
	}
}

type responseWithCode struct {
	Code int `json:"code"`
}

func (f *baseFetcher) GetUpdatedTokens() (string, string) {
	return f.accessToken, f.refreshToken
}

func (f *baseFetcher) Fetch(req *http.Request) ([]byte, error) {
	if f.accessToken != "" {
		req.Header.Set("Authorization", f.accessToken)
	}

	responseBytes, err := f.performRequest(req)
	if err != nil {
		return []byte{}, err
	}

	responseWithCode := responseWithCode{}
	err = json.Unmarshal(responseBytes, &responseWithCode)
	if err != nil {
		return []byte{}, err
	}

	if responseWithCode.Code == invalidTokenCode {
		accessToken, refreshToken, err := f.tokenRefresher.Refresh(f.refreshToken)
		if err != nil {
			return []byte{}, fmt.Errorf("Failed to refresh auth token: %s", err.Error())
		}

		f.accessToken = accessToken
		f.refreshToken = refreshToken

		req.Header.Set("Authorization", accessToken)

		return f.performRequest(req)
	}

	return responseBytes, nil
}

func (f *baseFetcher) performRequest(req *http.Request) ([]byte, error) {
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
