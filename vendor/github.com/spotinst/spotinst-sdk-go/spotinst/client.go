package spotinst

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Client struct {
	// HTTP client used to communicate with the Spotinst API.
	HttpClient *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent for client.
	UserAgent string

	//	Spotinst makes a call to an authorization API using your username and
	//	password, returning an 'Access Token' and a 'Refresh Token'.
	//	Our use case does not require the refresh token, but we should implement
	//	for completeness.
	AccessToken  string
	RefreshToken string

	// Services used for communicating with the API.
	Subscription *SubscriptionService
	AwsGroup     *AwsGroupService
}

// NewClient returns a new Spotinst API client.
func NewClient(creds *Credentials) (*Client, error) {
	baseURL, _ := url.Parse(apiURL)
	c := &Client{HttpClient: &http.Client{}, BaseURL: baseURL, UserAgent: userAgent}

	if creds != nil {
		if creds.Token != "" {
			// Use a Personal API Access Token.
			c.AccessToken = creds.Token
		} else {
			// Get new OAuth access and refresh tokens using the client credentials.
			accessToken, refreshToken, err := getOAuthTokens(creds.Email, creds.Password, creds.ClientID, creds.ClientSecret)
			if err != nil {
				return nil, err
			}

			c.AccessToken = accessToken
			c.RefreshToken = refreshToken
		}
	}

	// Spotinst services.
	c.Subscription = &SubscriptionService{client: c}
	c.AwsGroup = &AwsGroupService{client: c}

	return c, nil
}

// getOAuthTokens creates an Authorization request to get an access and refresh token.
func getOAuthTokens(username, password, clientID, clientSecret string) (string, string, error) {
	res, err := http.PostForm(
		fmt.Sprintf("%s/token", oauthURL),
		url.Values{
			"grant_type":    {"password"},
			"username":      {username},
			"password":      {password},
			"client_id":     {clientID},
			"client_secret": {clientSecret},
		},
	)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	err = CheckResponse(res)
	if err != nil {
		return "", "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", err
	}

	var resp Response
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return string(body), "JSON Decode Error", err
	}

	var accessToken, refreshToken string
	for _, i := range resp.Response.Items {
		m := i.(map[string]interface{})
		if v, ok := m["accessToken"].(string); ok {
			accessToken = v
		}
		if v, ok := m["refreshToken"].(string); ok {
			refreshToken = v
		}
	}

	return accessToken, refreshToken, err
}
