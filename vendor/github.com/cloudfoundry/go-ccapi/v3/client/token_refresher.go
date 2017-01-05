package client

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudfoundry/go-ccapi/v3/routing"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter -o fakes/fake_token_refresher.go . TokenRefresher
type TokenRefresher interface {
	Refresh(oldRefreshToken string) (string, string, error)
}

type tokenRefresher struct {
	uaaRequestGenerator *rata.RequestGenerator
	httpClient          *http.Client
}

func NewTokenRefresher(uaaEndpoint string) TokenRefresher {
	return &tokenRefresher{
		uaaRequestGenerator: rata.NewRequestGenerator(uaaEndpoint, routing.UAARoutes),
		httpClient:          &http.Client{},
	}
}

func (t tokenRefresher) Refresh(oldRefreshToken string) (string, string, error) {
	req, err := t.uaaRequestGenerator.CreateRequest("refresh_token", rata.Params{}, strings.NewReader(""))
	if err != nil {
		return "", "", err
	}

	data := url.Values{
		"refresh_token": {oldRefreshToken},
		"grant_type":    {"refresh_token"},
		"scope":         {""},
	}

	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("cf:")))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = data.Encode()

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	errorResponse := errorResponse{}
	err = json.Unmarshal(b, &errorResponse)
	if err != nil {
		return "", "", err
	}

	if errorResponse.Code != "" {
		return "", "", errors.New(errorResponse.Description)
	}

	authResponse := authResponse{}
	err = json.Unmarshal(b, &authResponse)
	if err != nil {
		return "", "", err
	}

	return authResponse.AccessToken, authResponse.RefreshToken, nil
}

type errorResponse struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}
