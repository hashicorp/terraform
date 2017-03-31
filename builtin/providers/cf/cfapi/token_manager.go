package cfapi

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/net"
)

// Note - This file was copied from github.com/cloudfoundry/cli/cf/api/authentication
// so the AuthManager can be extended with capabilities not present in CF CLI code

var errPreventRedirect = errors.New("prevent-redirect")

// AuthManager -
type AuthManager struct {
	config  coreconfig.ReadWriter
	gateway net.Gateway
	dumper  net.RequestDumper
}

// authenticationResponse -
type authenticationResponse struct {
	AccessToken  string           `json:"access_token"`
	TokenType    string           `json:"token_type"`
	RefreshToken string           `json:"refresh_token"`
	Error        uaaErrorResponse `json:"error"`
}

// NewAuthManager -
func NewAuthManager(gateway net.Gateway, config coreconfig.ReadWriter, dumper net.RequestDumper) *AuthManager {
	return &AuthManager{
		config:  config,
		gateway: gateway,
		dumper:  dumper,
	}
}

// DumpRequest -
func (tm *AuthManager) DumpRequest(req *http.Request) {
	tm.dumper.DumpRequest(req)
}

// DumpResponse -
func (tm *AuthManager) DumpResponse(res *http.Response) {
	tm.dumper.DumpResponse(res)
}

// Authorize -
func (tm *AuthManager) Authorize(token string) (string, error) {

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			tm.DumpRequest(req)
			return errPreventRedirect
		},
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: tm.config.IsSSLDisabled(),
			},
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	authorizeURL, err := url.Parse(tm.config.UaaEndpoint())
	if err != nil {
		return "", err
	}

	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("grant_type", "authorization_code")
	values.Set("client_id", tm.config.SSHOAuthClient())

	authorizeURL.Path = "/oauth/authorize"
	authorizeURL.RawQuery = values.Encode()

	authorizeReq, err := http.NewRequest("GET", authorizeURL.String(), nil)
	if err != nil {
		return "", err
	}

	authorizeReq.Header.Add("authorization", token)

	resp, err := httpClient.Do(authorizeReq)
	if resp != nil {
		tm.DumpResponse(resp)
	}
	if err == nil {
		return "", errors.New(i18n.T("Authorization server did not redirect with one time code"))
	}

	if netErr, ok := err.(*url.Error); !ok || netErr.Err != errPreventRedirect {
		return "", errors.New(i18n.T("Error requesting one time code from server: {{.Error}}", map[string]interface{}{"Error": err.Error()}))
	}

	loc, err := resp.Location()
	if err != nil {
		return "", errors.New(i18n.T("Error getting the redirected location: {{.Error}}", map[string]interface{}{"Error": err.Error()}))
	}

	codes := loc.Query()["code"]
	if len(codes) != 1 {
		return "", errors.New(i18n.T("Unable to acquire one time code from authorization response"))
	}

	return codes[0], nil
}

// Authenticate -
func (tm *AuthManager) Authenticate(credentials map[string]string) error {

	data := url.Values{
		"grant_type": {"password"},
		"scope":      {""},
	}
	for key, val := range credentials {
		data[key] = []string{val}
	}

	response, err := tm.getAuthToken("cf", "", data)
	if err != nil {
		httpError, ok := err.(errors.HTTPError)
		if ok {
			switch {
			case httpError.StatusCode() == http.StatusUnauthorized:
				return errors.New(i18n.T("Credentials were rejected, please try again."))
			case httpError.StatusCode() >= http.StatusInternalServerError:
				return errors.New(i18n.T("The targeted API endpoint could not be reached."))
			}
		}
		return err
	}

	tm.config.SetAccessToken(fmt.Sprintf("%s %s", response.TokenType, response.AccessToken))
	tm.config.SetRefreshToken(response.RefreshToken)
	return nil
}

// getClientToken -
func (tm *AuthManager) getClientToken(clientID, clientSecret string) (clientToken string, err error) {

	data := url.Values{
		"grant_type": {"client_credentials"},
	}

	response, err := tm.getAuthToken(clientID, clientSecret, data)
	if err != nil {
		httpError, ok := err.(errors.HTTPError)
		if ok {
			switch {
			case httpError.StatusCode() == http.StatusUnauthorized:
				err = errors.New(i18n.T("Credentials were rejected, please try again."))
			case httpError.StatusCode() >= http.StatusInternalServerError:
				err = errors.New(i18n.T("The targeted API endpoint could not be reached."))
			}
		}
		return
	}

	clientToken = fmt.Sprintf("%s %s", response.TokenType, response.AccessToken)
	return
}

// GetLoginPromptsAndSaveUAAServerURL -
func (tm *AuthManager) GetLoginPromptsAndSaveUAAServerURL() (prompts map[string]coreconfig.AuthPrompt, err error) {

	url := fmt.Sprintf("%s/login", tm.config.AuthenticationEndpoint())

	resource := &loginResource{}
	err = tm.gateway.GetResource(url, resource)
	if err != nil {
		return
	}

	prompts = resource.parsePrompts()
	if resource.Links["uaa"] == "" {
		tm.config.SetUaaEndpoint(tm.config.AuthenticationEndpoint())
	} else {
		tm.config.SetUaaEndpoint(resource.Links["uaa"])
	}
	return
}

// RefreshAuthToken -
func (tm *AuthManager) RefreshAuthToken() (string, error) {

	data := url.Values{
		"refresh_token": {tm.config.RefreshToken()},
		"grant_type":    {"refresh_token"},
		"scope":         {""},
	}

	response, err := tm.getAuthToken("cf", "", data)
	if err != nil {
		return "", err
	}

	tm.config.SetAccessToken(fmt.Sprintf("%s %s", response.TokenType, response.AccessToken))
	tm.config.SetRefreshToken(response.RefreshToken)

	return tm.config.AccessToken(), err
}

func (tm *AuthManager) getAuthToken(clientID, clientSecret string, data url.Values) (*authenticationResponse, error) {

	path := fmt.Sprintf("%s/oauth/token", tm.config.AuthenticationEndpoint())
	request, err := tm.gateway.NewRequest("POST", path,
		"Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)),
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%s: %s", i18n.T("Failed to start oauth request"), err.Error())
	}
	request.HTTPReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response := new(authenticationResponse)
	_, err = tm.gateway.PerformRequestForJSONResponse(request, &response)

	switch err.(type) {
	case nil:
	case errors.HTTPError:
		return nil, err
	case *errors.InvalidTokenError:
		return nil, errors.New(i18n.T("Authentication has expired.."))
	default:
		return nil, fmt.Errorf("%s: %s", i18n.T("auth request failed"), err.Error())
	}

	// TODO: get the actual status code
	if len(response.Error.Code) > 0 {
		return nil, errors.NewHTTPError(0, response.Error.Code, response.Error.Description)
	}

	return response, nil
}

type loginResource struct {
	Prompts map[string][]string
	Links   map[string]string
}

var knownAuthPromptTypes = map[string]coreconfig.AuthPromptType{
	"text":     coreconfig.AuthPromptTypeText,
	"password": coreconfig.AuthPromptTypePassword,
}

func (r *loginResource) parsePrompts() (prompts map[string]coreconfig.AuthPrompt) {
	prompts = make(map[string]coreconfig.AuthPrompt)
	for key, val := range r.Prompts {
		prompts[key] = coreconfig.AuthPrompt{
			Type:        knownAuthPromptTypes[val[0]],
			DisplayName: val[1],
		}
	}
	return
}
