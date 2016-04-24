package azure

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

var expirationBase time.Time

func init() {
	expirationBase, _ = time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")
}

type tokenRequester struct {
	clientID     string
	clientSecret string
	tenantID     string

	refreshWithin time.Duration

	httpClient *retryablehttp.Client

	l            sync.Mutex
	currentToken *token
}

func newTokenRequester(client *retryablehttp.Client, clientID, clientSecret, tenantID string) *tokenRequester {
	return &tokenRequester{
		clientID:      clientID,
		clientSecret:  clientSecret,
		tenantID:      tenantID,
		refreshWithin: 5 * time.Minute,
		httpClient:    client,
	}
}

// addAuthorizationToRequest adds an Authorization header to an http.Request, having ensured
// that the token is sufficiently fresh. This may invoke network calls, so should not be
// relied on to return quickly.
func (tr *tokenRequester) addAuthorizationToRequest(request *retryablehttp.Request) error {
	token, err := tr.getUsableToken()
	if err != nil {
		return fmt.Errorf("Error obtaining authorization token: %s", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
	return nil
}

func (tr *tokenRequester) getUsableToken() (*token, error) {
	tr.l.Lock()
	defer tr.l.Unlock()

	if tr.currentToken != nil && !tr.currentToken.willExpireIn(tr.refreshWithin) {
		return tr.currentToken, nil
	}

	newToken, err := tr.refreshToken()
	if err != nil {
		return nil, fmt.Errorf("Error refreshing token: %s", err)
	}

	tr.currentToken = newToken
	return newToken, nil
}

func (tr *tokenRequester) refreshToken() (*token, error) {
	oauthURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/%s?api-version=1.0", tr.tenantID, "token")

	v := url.Values{}
	v.Set("client_id", tr.clientID)
	v.Set("client_secret", tr.clientSecret)
	v.Set("grant_type", "client_credentials")
	v.Set("resource", "https://management.azure.com/")

	var newToken token
	response, err := tr.httpClient.PostForm(oauthURL, v)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&newToken)
	if err != nil {
		return nil, err
	}

	return &newToken, nil
}

// Token encapsulates the access token used to authorize Azure requests.
type token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	ExpiresOn   string `json:"expires_on"`
	NotBefore   string `json:"not_before"`
	Resource    string `json:"resource"`
	TokenType   string `json:"token_type"`
}

// willExpireIn returns true if the Token will expire after the passed time.Duration interval
// from now, false otherwise.
func (t token) willExpireIn(d time.Duration) bool {
	s, err := strconv.Atoi(t.ExpiresOn)
	if err != nil {
		s = -3600
	}
	expiryTime := expirationBase.Add(time.Duration(s) * time.Second).UTC()

	return !expiryTime.After(time.Now().Add(d))
}
